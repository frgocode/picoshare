package sqlite_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestCreateAndGetFolder(t *testing.T) {
	dataStore := test_sqlite.New(t)

	folder, err := dataStore.CreateFolder(picoshare.FolderName("Projects"), nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if got, want := folder.Name, picoshare.FolderName("Projects"); got != want {
		t.Errorf("name=%v, want=%v", got, want)
	}
	if folder.ParentID != nil {
		t.Errorf("parent ID=%v, want nil", folder.ParentID)
	}

	retrieved, err := dataStore.GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("failed to get folder: %v", err)
	}
	if got, want := retrieved.Name, picoshare.FolderName("Projects"); got != want {
		t.Errorf("name=%v, want=%v", got, want)
	}
}

func TestNestedFolderCreation(t *testing.T) {
	dataStore := test_sqlite.New(t)

	parent, err := dataStore.CreateFolder(picoshare.FolderName("Parent"), nil)
	if err != nil {
		t.Fatalf("failed to create parent folder: %v", err)
	}

	child, err := dataStore.CreateFolder(picoshare.FolderName("Child"), &parent.ID)
	if err != nil {
		t.Fatalf("failed to create child folder: %v", err)
	}
	if child.ParentID == nil || !child.ParentID.Equal(parent.ID) {
		t.Fatalf("child parent mismatch")
	}

	grandchild, err := dataStore.CreateFolder(picoshare.FolderName("Grandchild"), &child.ID)
	if err != nil {
		t.Fatalf("failed to create grandchild folder: %v", err)
	}

	path, err := dataStore.GetFolderPath(grandchild.ID)
	if err != nil {
		t.Fatalf("failed to get folder path: %v", err)
	}
	if got, want := len(path), 3; got != want {
		t.Fatalf("path length=%d, want=%d", got, want)
	}
	if got, want := path[0].Name, picoshare.FolderName("Parent"); got != want {
		t.Errorf("path[0]=%v, want=%v", got, want)
	}
	if got, want := path[2].Name, picoshare.FolderName("Grandchild"); got != want {
		t.Errorf("path[2]=%v, want=%v", got, want)
	}

	children, err := dataStore.ListChildFolders(&parent.ID)
	if err != nil {
		t.Fatalf("failed to list child folders: %v", err)
	}
	if got, want := len(children), 1; got != want {
		t.Fatalf("child count=%d, want=%d", got, want)
	}

	rootChildren, err := dataStore.ListChildFolders(nil)
	if err != nil {
		t.Fatalf("failed to list root folders: %v", err)
	}
	if got, want := len(rootChildren), 1; got != want {
		t.Fatalf("root child count=%d, want=%d", got, want)
	}
}

func TestDuplicateSiblingFolderNamesAreRejected(t *testing.T) {
	dataStore := test_sqlite.New(t)

	if _, err := dataStore.CreateFolder(picoshare.FolderName("Docs"), nil); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if _, err := dataStore.CreateFolder(picoshare.FolderName("Docs"), nil); err == nil {
		t.Fatalf("expected duplicate root folder name to fail")
	} else if _, ok := err.(store.DuplicateFolderNameError); !ok {
		t.Fatalf("expected DuplicateFolderNameError, got %T: %v", err, err)
	}

	parentA, err := dataStore.CreateFolder(picoshare.FolderName("A"), nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	parentB, err := dataStore.CreateFolder(picoshare.FolderName("B"), nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	if _, err := dataStore.CreateFolder(picoshare.FolderName("Same"), &parentA.ID); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if _, err := dataStore.CreateFolder(picoshare.FolderName("Same"), &parentB.ID); err != nil {
		t.Fatalf("same name in different parents should succeed, got: %v", err)
	}
	if _, err := dataStore.CreateFolder(picoshare.FolderName("Same"), &parentA.ID); err == nil {
		t.Fatalf("expected duplicate sibling name to fail")
	}
}

func TestInvalidParentReferencesAreRejected(t *testing.T) {
	dataStore := test_sqlite.New(t)

	bogus := picoshare.MustCreateFolderID("AAAAAAAAAA")
	if _, err := dataStore.CreateFolder(picoshare.FolderName("Orphan"), &bogus); err == nil {
		t.Fatalf("expected invalid parent to fail")
	}

	if _, err := dataStore.GetFolder(bogus); err == nil {
		t.Fatalf("expected missing folder to fail")
	} else if _, ok := err.(store.FolderNotFoundError); !ok {
		t.Fatalf("expected FolderNotFoundError, got %T", err)
	}
}

func TestFolderCyclePrevention(t *testing.T) {
	dataStore := test_sqlite.New(t)

	parent, _ := dataStore.CreateFolder(picoshare.FolderName("Parent"), nil)
	child, _ := dataStore.CreateFolder(picoshare.FolderName("Child"), &parent.ID)
	grandchild, _ := dataStore.CreateFolder(picoshare.FolderName("Grandchild"), &child.ID)

	if err := dataStore.MoveFolder(parent.ID, &grandchild.ID); err == nil {
		t.Fatalf("moving parent into grandchild should fail")
	} else if _, ok := err.(store.FolderCycleError); !ok {
		t.Fatalf("expected FolderCycleError, got %T: %v", err, err)
	}

	if err := dataStore.MoveFolder(parent.ID, &parent.ID); err == nil {
		t.Fatalf("moving folder into itself should fail")
	}
}

func TestRenameAndMoveFolder(t *testing.T) {
	dataStore := test_sqlite.New(t)

	a, _ := dataStore.CreateFolder(picoshare.FolderName("A"), nil)
	b, _ := dataStore.CreateFolder(picoshare.FolderName("B"), nil)
	c, _ := dataStore.CreateFolder(picoshare.FolderName("C"), &a.ID)

	if err := dataStore.RenameFolder(c.ID, picoshare.FolderName("C-renamed")); err != nil {
		t.Fatalf("failed to rename folder: %v", err)
	}
	renamed, _ := dataStore.GetFolder(c.ID)
	if got, want := renamed.Name, picoshare.FolderName("C-renamed"); got != want {
		t.Errorf("name=%v, want=%v", got, want)
	}

	if _, err := dataStore.CreateFolder(picoshare.FolderName("D"), &a.ID); err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := dataStore.RenameFolder(c.ID, picoshare.FolderName("D")); err == nil {
		t.Fatalf("renaming to duplicate sibling name should fail")
	}

	if err := dataStore.MoveFolder(c.ID, &b.ID); err != nil {
		t.Fatalf("failed to move folder: %v", err)
	}
	moved, _ := dataStore.GetFolder(c.ID)
	if moved.ParentID == nil || !moved.ParentID.Equal(b.ID) {
		t.Fatalf("move did not update parent")
	}

	if err := dataStore.MoveFolder(c.ID, nil); err != nil {
		t.Fatalf("failed to move folder to root: %v", err)
	}
	backToRoot, _ := dataStore.GetFolder(c.ID)
	if backToRoot.ParentID != nil {
		t.Fatalf("expected folder to be at root")
	}
}

func TestDeleteFolderRules(t *testing.T) {
	dataStore := test_sqlite.New(t)

	parent, _ := dataStore.CreateFolder(picoshare.FolderName("Parent"), nil)
	if _, err := dataStore.CreateFolder(picoshare.FolderName("Child"), &parent.ID); err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	if err := dataStore.DeleteFolder(parent.ID); err == nil {
		t.Fatalf("deleting nonempty folder should fail")
	} else if _, ok := err.(store.FolderNotEmptyError); !ok {
		t.Fatalf("expected FolderNotEmptyError, got %T: %v", err, err)
	}

	childFolders, _ := dataStore.ListChildFolders(&parent.ID)
	if err := dataStore.DeleteFolder(childFolders[0].ID); err != nil {
		t.Fatalf("failed to delete empty child folder: %v", err)
	}
	if err := dataStore.DeleteFolder(parent.ID); err != nil {
		t.Fatalf("failed to delete now-empty parent folder: %v", err)
	}
}

func TestFilePlacementInFolders(t *testing.T) {
	dataStore := test_sqlite.New(t)

	data := "dummy data"
	entryID := picoshare.MustCreateEntryID("AAAAAAAAAA")
	if err := dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "file.txt",
		Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(data)),
	}); err != nil {
		t.Fatalf("failed to insert entry: %v", err)
	}

	rootFiles, err := dataStore.GetEntriesMetadataInFolder(nil)
	if err != nil {
		t.Fatalf("failed to list root files: %v", err)
	}
	if got, want := len(rootFiles), 1; got != want {
		t.Fatalf("root file count=%d, want=%d", got, want)
	}

	folder, _ := dataStore.CreateFolder(picoshare.FolderName("Docs"), nil)
	if err := dataStore.SetFileFolder(entryID, &folder.ID); err != nil {
		t.Fatalf("failed to move file to folder: %v", err)
	}

	rootFiles, _ = dataStore.GetEntriesMetadataInFolder(nil)
	if got, want := len(rootFiles), 0; got != want {
		t.Fatalf("root file count after move=%d, want=%d", got, want)
	}
	folderFiles, _ := dataStore.GetEntriesMetadataInFolder(&folder.ID)
	if got, want := len(folderFiles), 1; got != want {
		t.Fatalf("folder file count=%d, want=%d", got, want)
	}

	current, err := dataStore.GetFileFolder(entryID)
	if err != nil {
		t.Fatalf("failed to get file folder: %v", err)
	}
	if current == nil || !current.ID.Equal(folder.ID) {
		t.Fatalf("file folder mismatch")
	}

	if err := dataStore.SetFileFolder(entryID, nil); err != nil {
		t.Fatalf("failed to move file to root: %v", err)
	}
	current, _ = dataStore.GetFileFolder(entryID)
	if current != nil {
		t.Fatalf("expected file to be in root")
	}
}

func TestFileDeletionCleansFolderAssociation(t *testing.T) {
	dataStore := test_sqlite.New(t)

	data := "dummy data"
	entryID := picoshare.MustCreateEntryID("BBBBBBBBBB")
	dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "file.txt",
		Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(data)),
	})
	folder, _ := dataStore.CreateFolder(picoshare.FolderName("Docs"), nil)
	dataStore.SetFileFolder(entryID, &folder.ID)

	if err := dataStore.DeleteEntry(entryID); err != nil {
		t.Fatalf("failed to delete entry: %v", err)
	}

	current, err := dataStore.GetFileFolder(entryID)
	if err != nil {
		t.Fatalf("failed to get file folder: %v", err)
	}
	if current != nil {
		t.Fatalf("expected orphan association to be removed")
	}

	if err := dataStore.DeleteFolder(folder.ID); err != nil {
		t.Fatalf("folder should be empty after file deletion, got: %v", err)
	}
}

func TestExpirationCleanupRemovesFolderAssociations(t *testing.T) {
	dataStore := test_sqlite.New(t)

	data := "dummy data"
	entryID := picoshare.MustCreateEntryID("CCCCCCCCCC")
	dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "file.txt",
		Uploaded: mustParseTime("2024-01-01T00:00:00Z"),
		Expires:  mustParseExpirationTime("2024-06-01T00:00:00Z"),
		Size:     mustParseFileSize(len(data)),
	})
	folder, _ := dataStore.CreateFolder(picoshare.FolderName("Docs"), nil)
	dataStore.SetFileFolder(entryID, &folder.ID)

	if err := dataStore.Purge(); err != nil {
		t.Fatalf("failed to purge: %v", err)
	}

	folderFiles, _ := dataStore.GetEntriesMetadataInFolder(&folder.ID)
	if got, want := len(folderFiles), 0; got != want {
		t.Fatalf("expired file should be gone from folder, count=%d, want=%d", got, want)
	}
}

func TestOneFolderPerFile(t *testing.T) {
	dataStore := test_sqlite.New(t)

	data := "dummy data"
	entryID := picoshare.MustCreateEntryID("DDDDDDDDDD")
	dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
		ID:       entryID,
		Filename: "file.txt",
		Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
		Size:     mustParseFileSize(len(data)),
	})

	folderA, _ := dataStore.CreateFolder(picoshare.FolderName("A"), nil)
	folderB, _ := dataStore.CreateFolder(picoshare.FolderName("B"), nil)

	dataStore.SetFileFolder(entryID, &folderA.ID)
	dataStore.SetFileFolder(entryID, &folderB.ID)

	current, _ := dataStore.GetFileFolder(entryID)
	if current == nil || !current.ID.Equal(folderB.ID) {
		t.Fatalf("expected file to be in folder B after second move")
	}

	inA, _ := dataStore.GetEntriesMetadataInFolder(&folderA.ID)
	if got, want := len(inA), 0; got != want {
		t.Fatalf("folder A should be empty, count=%d, want=%d", got, want)
	}
}

func TestExistingFilesAppearInRoot(t *testing.T) {
	dataStore := test_sqlite.New(t)

	for _, id := range []string{"EEEEEEEEEE", "FFFFFFFFFF"} {
		data := "content " + id
		dataStore.InsertEntry(strings.NewReader(data), picoshare.UploadMetadata{
			ID:       picoshare.MustCreateEntryID(id),
			Filename: picoshare.Filename("file-" + id + ".txt"),
			Uploaded: mustParseTime("2025-05-25T00:00:00Z"),
			Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
			Size:     mustParseFileSize(len(data)),
		})
	}

	rootFiles, err := dataStore.GetEntriesMetadataInFolder(nil)
	if err != nil {
		t.Fatalf("failed to list root files: %v", err)
	}
	if got, want := len(rootFiles), 2; got != want {
		t.Fatalf("existing files should appear in root, count=%d, want=%d", got, want)
	}
}
