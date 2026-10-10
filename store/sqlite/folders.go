package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

func (s Store) CreateFolder(name picoshare.FolderName, parentID *picoshare.FolderID) (picoshare.Folder, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return picoshare.Folder{}, err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback create folder: %v", err)
		}
	}()

	if parentID != nil {
		if err := checkFolderExistsTx(tx, *parentID); err != nil {
			return picoshare.Folder{}, err
		}
	}

	if err := checkDuplicateFolderNameTx(tx, string(name), parentID, nil); err != nil {
		return picoshare.Folder{}, err
	}

	id := picoshare.NewFolderID()
	created := s.now()
	if created.IsZero() {
		return picoshare.Folder{}, fmt.Errorf("failed to create folder: invalid time")
	}

	var parentRaw any
	if parentID != nil {
		parentRaw = parentID.String()
	}

	if _, err := tx.Exec(`
	INSERT INTO virtual_folders (id, name, parent_id, entry_id, created_time)
	VALUES (:id, :name, :parent_id, NULL, :created_time)`,
		sql.Named("id", id.String()),
		sql.Named("name", string(name)),
		sql.Named("parent_id", parentRaw),
		sql.Named("created_time", formatTime(created))); err != nil {
		if isDuplicateFolderNameError(err) {
			return picoshare.Folder{}, store.DuplicateFolderNameError{
				Name: store.FolderNameDescription{Name: string(name)},
			}
		}
		return picoshare.Folder{}, err
	}

	if err := tx.Commit(); err != nil {
		return picoshare.Folder{}, err
	}

	return picoshare.Folder{
		ID:       id,
		Name:     name,
		ParentID: parentID,
		Created:  created,
	}, nil
}

func (s Store) GetFolder(id picoshare.FolderID) (picoshare.Folder, error) {
	var name string
	var parentRaw sql.NullString
	var createdRaw string
	err := s.db.QueryRow(`
	SELECT name, parent_id, created_time
	FROM virtual_folders
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("id", id.String())).Scan(&name, &parentRaw, &createdRaw)
	if err == sql.ErrNoRows {
		return picoshare.Folder{}, store.FolderNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.Folder{}, err
	}

	created, err := parseDatetime(createdRaw)
	if err != nil {
		return picoshare.Folder{}, err
	}

	var parentID *picoshare.FolderID
	if parentRaw.Valid {
		parsed, err := picoshare.FolderIDFromString(parentRaw.String)
		if err != nil {
			return picoshare.Folder{}, fmt.Errorf("failed to parse parent folder ID from database: %w", err)
		}
		parentID = &parsed
	}

	return picoshare.Folder{
		ID:       id,
		Name:     picoshare.FolderName(name),
		ParentID: parentID,
		Created:  created,
	}, nil
}

func (s Store) ListChildFolders(parentID *picoshare.FolderID) ([]picoshare.Folder, error) {
	var rows *sql.Rows
	var err error
	if parentID == nil {
		rows, err = s.db.Query(`
		SELECT id, name, parent_id, created_time
		FROM virtual_folders
		WHERE parent_id IS NULL AND entry_id IS NULL
		ORDER BY name ASC`)
	} else {
		rows, err = s.db.Query(`
		SELECT id, name, parent_id, created_time
		FROM virtual_folders
		WHERE parent_id = :parent_id AND entry_id IS NULL
		ORDER BY name ASC`,
			sql.Named("parent_id", parentID.String()))
	}
	if err != nil {
		return []picoshare.Folder{}, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("failed to close rows: %v", cerr)
		}
	}()

	return foldersFromRows(rows)
}

func (s Store) ListAllFolders() ([]picoshare.Folder, error) {
	rows, err := s.db.Query(`
	SELECT id, name, parent_id, created_time
	FROM virtual_folders
	WHERE entry_id IS NULL
	ORDER BY name ASC`)
	if err != nil {
		return []picoshare.Folder{}, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("failed to close rows: %v", cerr)
		}
	}()

	return foldersFromRows(rows)
}

func (s Store) GetFolderPath(id picoshare.FolderID) ([]picoshare.Folder, error) {
	path := []picoshare.Folder{}
	currentID := id
	for i := 0; i < 100; i++ {
		folder, err := s.GetFolder(currentID)
		if err != nil {
			return []picoshare.Folder{}, err
		}
		path = append([]picoshare.Folder{folder}, path...)
		if folder.ParentID == nil {
			return path, nil
		}
		currentID = *folder.ParentID
	}

	return []picoshare.Folder{}, fmt.Errorf("failed to resolve folder path: hierarchy too deep")
}

func (s Store) RenameFolder(id picoshare.FolderID, name picoshare.FolderName) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback rename folder: %v", err)
		}
	}()

	parentID, err := getFolderParentTx(tx, id)
	if err != nil {
		return err
	}

	if err := checkDuplicateFolderNameTx(tx, string(name), parentID, &id); err != nil {
		return err
	}

	res, err := tx.Exec(`
	UPDATE virtual_folders
	SET name = :name
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("name", string(name)),
		sql.Named("id", id.String()))
	if err != nil {
		if isDuplicateFolderNameError(err) {
			return store.DuplicateFolderNameError{
				Name: store.FolderNameDescription{Name: string(name)},
			}
		}
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return store.FolderNotFoundError{ID: id}
	}

	return tx.Commit()
}

func (s Store) MoveFolder(id picoshare.FolderID, newParentID *picoshare.FolderID) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback move folder: %v", err)
		}
	}()

	folder, err := getFolderTx(tx, id)
	if err != nil {
		return err
	}

	if newParentID != nil {
		if newParentID.Equal(id) {
			return store.FolderCycleError{ID: id}
		}

		if err := checkFolderExistsTx(tx, *newParentID); err != nil {
			return err
		}

		ancestors, err := getAncestorIDsTx(tx, *newParentID)
		if err != nil {
			return err
		}
		for _, ancestor := range ancestors {
			if ancestor.Equal(id) {
				return store.FolderCycleError{ID: id}
			}
		}
	}

	if err := checkDuplicateFolderNameTx(tx, string(folder.Name), newParentID, &id); err != nil {
		return err
	}

	var parentRaw any
	if newParentID != nil {
		parentRaw = newParentID.String()
	}

	res, err := tx.Exec(`
	UPDATE virtual_folders
	SET parent_id = :parent_id
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("parent_id", parentRaw),
		sql.Named("id", id.String()))
	if err != nil {
		if isDuplicateFolderNameError(err) {
			return store.DuplicateFolderNameError{
				Name: store.FolderNameDescription{Name: string(folder.Name)},
			}
		}
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return store.FolderNotFoundError{ID: id}
	}

	return tx.Commit()
}

func (s Store) DeleteFolder(id picoshare.FolderID) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback delete folder: %v", err)
		}
	}()

	if err := checkFolderExistsTx(tx, id); err != nil {
		return err
	}

	var childCount int
	if err := tx.QueryRow(`
	SELECT COUNT(*)
	FROM virtual_folders
	WHERE parent_id = :id`,
		sql.Named("id", id.String())).Scan(&childCount); err != nil {
		return err
	}
	if childCount > 0 {
		return store.FolderNotEmptyError{ID: id}
	}

	res, err := tx.Exec(`
	DELETE FROM virtual_folders
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("id", id.String()))
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return store.FolderNotFoundError{ID: id}
	}

	return tx.Commit()
}

func (s Store) SetFileFolder(entryID picoshare.EntryID, folderID *picoshare.FolderID) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback set file folder: %v", err)
		}
	}()

	var entryExists int
	if err := tx.QueryRow(`
	SELECT COUNT(*)
	FROM entries
	WHERE id = :entry_id`,
		sql.Named("entry_id", entryID.String())).Scan(&entryExists); err != nil {
		return err
	}
	if entryExists == 0 {
		return store.EntryNotFoundError{ID: entryID}
	}

	if folderID == nil {
		if _, err := tx.Exec(`
		DELETE FROM virtual_folders
		WHERE entry_id = :entry_id`,
			sql.Named("entry_id", entryID.String())); err != nil {
			return err
		}

		return tx.Commit()
	}

	if err := checkFolderExistsTx(tx, *folderID); err != nil {
		return err
	}

	res, err := tx.Exec(`
	UPDATE virtual_folders
	SET parent_id = :parent_id
	WHERE entry_id = :entry_id`,
		sql.Named("parent_id", folderID.String()),
		sql.Named("entry_id", entryID.String()))
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return tx.Commit()
	}

	if _, err := tx.Exec(`
	INSERT INTO virtual_folders (id, name, parent_id, entry_id, created_time)
	VALUES (:id, NULL, :parent_id, :entry_id, :created_time)`,
		sql.Named("id", entryID.String()),
		sql.Named("parent_id", folderID.String()),
		sql.Named("entry_id", entryID.String()),
		sql.Named("created_time", formatTime(s.now()))); err != nil {
		return err
	}

	return tx.Commit()
}

func (s Store) GetFileFolder(entryID picoshare.EntryID) (*picoshare.Folder, error) {
	var parentRaw string
	err := s.db.QueryRow(`
	SELECT parent_id
	FROM virtual_folders
	WHERE entry_id = :entry_id`,
		sql.Named("entry_id", entryID.String())).Scan(&parentRaw)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	folderID, err := picoshare.FolderIDFromString(parentRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse folder ID from database: %w", err)
	}

	folder, err := s.GetFolder(folderID)
	if err != nil {
		return nil, err
	}

	return &folder, nil
}

func (s Store) GetEntriesMetadataInFolder(folderID *picoshare.FolderID) ([]picoshare.UploadMetadata, error) {
	var rows *sql.Rows
	var err error
	if folderID == nil {
		rows, err = s.db.Query(`
		SELECT
			entries.id AS id,
			entries.filename AS filename,
			entries.note AS note,
			entries.content_type AS content_type,
			entries.upload_time AS upload_time,
			entries.expiration_time AS expiration_time,
			sizes.file_size AS file_size
		FROM
			entries
		INNER JOIN
			(
				SELECT
					id,
					SUM(LENGTH(chunk)) AS file_size
				FROM
					entries_data
				GROUP BY
					id
			) sizes ON entries.id = sizes.id
		LEFT JOIN
			virtual_folders ON entries.id = virtual_folders.entry_id
		WHERE
			virtual_folders.entry_id IS NULL`)
	} else {
		rows, err = s.db.Query(`
		SELECT
			entries.id AS id,
			entries.filename AS filename,
			entries.note AS note,
			entries.content_type AS content_type,
			entries.upload_time AS upload_time,
			entries.expiration_time AS expiration_time,
			sizes.file_size AS file_size
		FROM
			entries
		INNER JOIN
			(
				SELECT
					id,
					SUM(LENGTH(chunk)) AS file_size
				FROM
					entries_data
				GROUP BY
					id
			) sizes ON entries.id = sizes.id
		INNER JOIN
			virtual_folders ON entries.id = virtual_folders.entry_id
		WHERE
			virtual_folders.parent_id = :folder_id`,
			sql.Named("folder_id", folderID.String()))
	}
	if err != nil {
		return []picoshare.UploadMetadata{}, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("failed to close rows: %v", cerr)
		}
	}()

	return entriesMetadataFromRows(rows)
}

func foldersFromRows(rows *sql.Rows) ([]picoshare.Folder, error) {
	folders := []picoshare.Folder{}
	for rows.Next() {
		var idRaw string
		var name string
		var parentRaw sql.NullString
		var createdRaw string
		if err := rows.Scan(&idRaw, &name, &parentRaw, &createdRaw); err != nil {
			return []picoshare.Folder{}, err
		}

		id, err := picoshare.FolderIDFromString(idRaw)
		if err != nil {
			return []picoshare.Folder{}, fmt.Errorf("failed to parse folder ID from database: %w", err)
		}

		created, err := parseDatetime(createdRaw)
		if err != nil {
			return []picoshare.Folder{}, err
		}

		var parentID *picoshare.FolderID
		if parentRaw.Valid {
			parsed, err := picoshare.FolderIDFromString(parentRaw.String)
			if err != nil {
				return []picoshare.Folder{}, fmt.Errorf("failed to parse parent folder ID from database: %w", err)
			}
			parentID = &parsed
		}

		folders = append(folders, picoshare.Folder{
			ID:       id,
			Name:     picoshare.FolderName(name),
			ParentID: parentID,
			Created:  created,
		})
	}

	return folders, nil
}

func entriesMetadataFromRows(rows *sql.Rows) ([]picoshare.UploadMetadata, error) {
	ee := []picoshare.UploadMetadata{}
	for rows.Next() {
		var id string
		var filename string
		var note *string
		var contentType string
		var uploadTimeRaw string
		var expirationTimeRaw string
		var fileSizeRaw uint64
		if err := rows.Scan(&id, &filename, &note, &contentType, &uploadTimeRaw, &expirationTimeRaw, &fileSizeRaw); err != nil {
			return []picoshare.UploadMetadata{}, err
		}
		entryID, err := picoshare.EntryIDFromString(id)
		if err != nil {
			return []picoshare.UploadMetadata{}, fmt.Errorf("failed to parse entry ID from database: %w", err)
		}

		ut, err := parseDatetime(uploadTimeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		et, err := parseDatetime(expirationTimeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		fileSize, err := picoshare.FileSizeFromUint64(fileSizeRaw)
		if err != nil {
			return []picoshare.UploadMetadata{}, err
		}

		ee = append(ee, picoshare.UploadMetadata{
			ID:          entryID,
			Filename:    picoshare.Filename(filename),
			Note:        picoshare.FileNote{Value: note},
			ContentType: picoshare.ContentType(contentType),
			Uploaded:    ut,
			Expires:     picoshare.ExpirationTime(et),
			Size:        fileSize,
		})
	}

	return ee, nil
}

func checkFolderExistsTx(tx *sql.Tx, id picoshare.FolderID) error {
	var count int
	if err := tx.QueryRow(`
	SELECT COUNT(*)
	FROM virtual_folders
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("id", id.String())).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return store.FolderNotFoundError{ID: id}
	}

	return nil
}

func getFolderTx(tx *sql.Tx, id picoshare.FolderID) (picoshare.Folder, error) {
	var name string
	var parentRaw sql.NullString
	var createdRaw string
	err := tx.QueryRow(`
	SELECT name, parent_id, created_time
	FROM virtual_folders
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("id", id.String())).Scan(&name, &parentRaw, &createdRaw)
	if err == sql.ErrNoRows {
		return picoshare.Folder{}, store.FolderNotFoundError{ID: id}
	} else if err != nil {
		return picoshare.Folder{}, err
	}

	created, err := parseDatetime(createdRaw)
	if err != nil {
		return picoshare.Folder{}, err
	}

	var parentID *picoshare.FolderID
	if parentRaw.Valid {
		parsed, err := picoshare.FolderIDFromString(parentRaw.String)
		if err != nil {
			return picoshare.Folder{}, fmt.Errorf("failed to parse parent folder ID from database: %w", err)
		}
		parentID = &parsed
	}

	return picoshare.Folder{
		ID:       id,
		Name:     picoshare.FolderName(name),
		ParentID: parentID,
		Created:  created,
	}, nil
}

func getFolderParentTx(tx *sql.Tx, id picoshare.FolderID) (*picoshare.FolderID, error) {
	var parentRaw sql.NullString
	err := tx.QueryRow(`
	SELECT parent_id
	FROM virtual_folders
	WHERE id = :id AND entry_id IS NULL`,
		sql.Named("id", id.String())).Scan(&parentRaw)
	if err == sql.ErrNoRows {
		return nil, store.FolderNotFoundError{ID: id}
	} else if err != nil {
		return nil, err
	}

	if !parentRaw.Valid {
		return nil, nil
	}

	parentID, err := picoshare.FolderIDFromString(parentRaw.String)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent folder ID from database: %w", err)
	}

	return &parentID, nil
}

func getAncestorIDsTx(tx *sql.Tx, id picoshare.FolderID) ([]picoshare.FolderID, error) {
	ancestors := []picoshare.FolderID{}
	currentID := id
	for i := 0; i < 100; i++ {
		var parentRaw sql.NullString
		err := tx.QueryRow(`
		SELECT parent_id
		FROM virtual_folders
		WHERE id = :id AND entry_id IS NULL`,
			sql.Named("id", currentID.String())).Scan(&parentRaw)
		if err == sql.ErrNoRows {
			return nil, store.FolderNotFoundError{ID: currentID}
		} else if err != nil {
			return nil, err
		}

		if !parentRaw.Valid {
			return ancestors, nil
		}

		parentID, err := picoshare.FolderIDFromString(parentRaw.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent folder ID from database: %w", err)
		}

		ancestors = append(ancestors, parentID)
		currentID = parentID
	}

	return nil, fmt.Errorf("failed to resolve folder ancestors: hierarchy too deep")
}

func checkDuplicateFolderNameTx(tx *sql.Tx, name string, parentID *picoshare.FolderID, excludeID *picoshare.FolderID) error {
	var count int
	var err error
	if parentID == nil {
		if excludeID == nil {
			err = tx.QueryRow(`
			SELECT COUNT(*)
			FROM virtual_folders
			WHERE parent_id IS NULL AND entry_id IS NULL AND name = :name`,
				sql.Named("name", name)).Scan(&count)
		} else {
			err = tx.QueryRow(`
			SELECT COUNT(*)
			FROM virtual_folders
			WHERE parent_id IS NULL AND entry_id IS NULL AND name = :name AND id != :exclude_id`,
				sql.Named("name", name),
				sql.Named("exclude_id", excludeID.String())).Scan(&count)
		}
	} else {
		if excludeID == nil {
			err = tx.QueryRow(`
			SELECT COUNT(*)
			FROM virtual_folders
			WHERE parent_id = :parent_id AND entry_id IS NULL AND name = :name`,
				sql.Named("parent_id", parentID.String()),
				sql.Named("name", name)).Scan(&count)
		} else {
			err = tx.QueryRow(`
			SELECT COUNT(*)
			FROM virtual_folders
			WHERE parent_id = :parent_id AND entry_id IS NULL AND name = :name AND id != :exclude_id`,
				sql.Named("parent_id", parentID.String()),
				sql.Named("name", name),
				sql.Named("exclude_id", excludeID.String())).Scan(&count)
		}
	}
	if err != nil {
		return err
	}
	if count > 0 {
		return store.DuplicateFolderNameError{
			Name: store.FolderNameDescription{Name: name},
		}
	}

	return nil
}

func isDuplicateFolderNameError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "virtual_folders")
}
