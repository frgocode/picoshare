package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

type (
	folderCreateRequest struct {
		Name     picoshare.FolderName
		ParentID *picoshare.FolderID
	}

	folderRenameRequest struct {
		ID   picoshare.FolderID
		Name picoshare.FolderName
	}

	folderMoveRequest struct {
		ID          picoshare.FolderID
		NewParentID *picoshare.FolderID
	}

	fileFolderMoveRequest struct {
		EntryID  picoshare.EntryID
		FolderID *picoshare.FolderID
	}

	FolderPostResponse struct {
		ID string `json:"id"`
	}
)

func parseFolderCreateRequest(r *http.Request) (folderCreateRequest, error) {
	var payload struct {
		Name     string `json:"name"`
		ParentID string `json:"parentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return folderCreateRequest{}, err
	}

	name, err := parse.FolderName(payload.Name)
	if err != nil {
		return folderCreateRequest{}, err
	}

	var parentID *picoshare.FolderID
	if payload.ParentID != "" {
		parsed, err := picoshare.FolderIDFromString(payload.ParentID)
		if err != nil {
			return folderCreateRequest{}, err
		}
		parentID = &parsed
	}

	return folderCreateRequest{
		Name:     name,
		ParentID: parentID,
	}, nil
}

func parseFolderRenameRequest(r *http.Request) (folderRenameRequest, error) {
	id, err := picoshare.FolderIDFromString(mux.Vars(r)["id"])
	if err != nil {
		return folderRenameRequest{}, err
	}

	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return folderRenameRequest{}, err
	}

	name, err := parse.FolderName(payload.Name)
	if err != nil {
		return folderRenameRequest{}, err
	}

	return folderRenameRequest{
		ID:   id,
		Name: name,
	}, nil
}

func parseFolderMoveRequest(r *http.Request) (folderMoveRequest, error) {
	id, err := picoshare.FolderIDFromString(mux.Vars(r)["id"])
	if err != nil {
		return folderMoveRequest{}, err
	}

	var payload struct {
		ParentID *string `json:"parentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return folderMoveRequest{}, err
	}

	var newParentID *picoshare.FolderID
	if payload.ParentID != nil && *payload.ParentID != "" {
		parsed, err := picoshare.FolderIDFromString(*payload.ParentID)
		if err != nil {
			return folderMoveRequest{}, err
		}
		newParentID = &parsed
	}

	return folderMoveRequest{
		ID:          id,
		NewParentID: newParentID,
	}, nil
}

func parseFileFolderMoveRequest(r *http.Request) (fileFolderMoveRequest, error) {
	entryID, err := picoshare.EntryIDFromString(mux.Vars(r)["id"])
	if err != nil {
		return fileFolderMoveRequest{}, err
	}

	var payload struct {
		FolderID *string `json:"folderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return fileFolderMoveRequest{}, err
	}

	var folderID *picoshare.FolderID
	if payload.FolderID != nil && *payload.FolderID != "" {
		parsed, err := picoshare.FolderIDFromString(*payload.FolderID)
		if err != nil {
			return fileFolderMoveRequest{}, err
		}
		folderID = &parsed
	}

	return fileFolderMoveRequest{
		EntryID:  entryID,
		FolderID: folderID,
	}, nil
}

func parseFolderIDFromQuery(r *http.Request, key string) (*picoshare.FolderID, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}

	id, err := picoshare.FolderIDFromString(raw)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

func (s Server) folderPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseFolderCreateRequest(r)
		if err != nil {
			log.Printf("invalid folder create request: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		folder, err := s.store.CreateFolder(req.Name, req.ParentID)
		if _, ok := errors.AsType[store.DuplicateFolderNameError](err); ok {
			http.Error(w, fmt.Sprintf("A folder with that name already exists: %v", err), http.StatusConflict)
			return
		} else if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, fmt.Sprintf("Parent folder not found: %v", err), http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("failed to create folder: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create folder: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, FolderPostResponse{ID: folder.ID.String()})
	}
}

func (s Server) folderPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseFolderRenameRequest(r)
		if err != nil {
			log.Printf("invalid folder rename request: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		err = s.store.RenameFolder(req.ID, req.Name)
		if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, "Folder not found", http.StatusNotFound)
			return
		} else if _, ok := errors.AsType[store.DuplicateFolderNameError](err); ok {
			http.Error(w, fmt.Sprintf("A folder with that name already exists: %v", err), http.StatusConflict)
			return
		} else if err != nil {
			log.Printf("failed to rename folder %v: %v", req.ID, err)
			http.Error(w, fmt.Sprintf("Failed to rename folder: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) folderMovePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseFolderMoveRequest(r)
		if err != nil {
			log.Printf("invalid folder move request: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		err = s.store.MoveFolder(req.ID, req.NewParentID)
		if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, "Folder not found", http.StatusNotFound)
			return
		} else if _, ok := errors.AsType[store.DuplicateFolderNameError](err); ok {
			http.Error(w, fmt.Sprintf("A folder with that name already exists in the destination: %v", err), http.StatusConflict)
			return
		} else if _, ok := errors.AsType[store.FolderCycleError](err); ok {
			http.Error(w, fmt.Sprintf("Cannot move folder there: %v", err), http.StatusBadRequest)
			return
		} else if _, ok := errors.AsType[store.InvalidFolderParentError](err); ok {
			http.Error(w, fmt.Sprintf("Invalid destination folder: %v", err), http.StatusBadRequest)
			return
		} else if err != nil {
			log.Printf("failed to move folder %v: %v", req.ID, err)
			http.Error(w, fmt.Sprintf("Failed to move folder: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) folderDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := picoshare.FolderIDFromString(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing folder ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid folder ID: %v", err), http.StatusBadRequest)
			return
		}

		err = s.store.DeleteFolder(id)
		if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, "Folder not found", http.StatusNotFound)
			return
		} else if _, ok := errors.AsType[store.FolderNotEmptyError](err); ok {
			http.Error(w, "Folder is not empty", http.StatusConflict)
			return
		} else if err != nil {
			log.Printf("failed to delete folder %v: %v", id, err)
			http.Error(w, fmt.Sprintf("Failed to delete folder: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func (s Server) folderListGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["parentId"]; ok {
			parentID, err := parseFolderIDFromQuery(r, "parentId")
			if err != nil {
				log.Printf("invalid parent folder ID: %v", err)
				http.Error(w, fmt.Sprintf("Invalid parent folder ID: %v", err), http.StatusBadRequest)
				return
			}

			if parentID != nil {
				if _, err := s.store.GetFolder(*parentID); err != nil {
					if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
						http.Error(w, "Parent folder not found", http.StatusNotFound)
						return
					}
					log.Printf("failed to retrieve parent folder: %v", err)
					http.Error(w, fmt.Sprintf("Failed to retrieve folders: %v", err), http.StatusInternalServerError)
					return
				}
			}

			folders, err := s.store.ListChildFolders(parentID)
			if err != nil {
				log.Printf("failed to list child folders: %v", err)
				http.Error(w, fmt.Sprintf("Failed to list folders: %v", err), http.StatusInternalServerError)
				return
			}

			respondJSON(w, foldersToResponse(folders))
			return
		}

		folders, err := s.store.ListAllFolders()
		if err != nil {
			log.Printf("failed to list folders: %v", err)
			http.Error(w, fmt.Sprintf("Failed to list folders: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, foldersToResponse(folders))
	}
}

func (s Server) folderGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := picoshare.FolderIDFromString(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing folder ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid folder ID: %v", err), http.StatusBadRequest)
			return
		}

		folder, err := s.store.GetFolder(id)
		if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, "Folder not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("failed to retrieve folder %v: %v", id, err)
			http.Error(w, fmt.Sprintf("Failed to retrieve folder: %v", err), http.StatusInternalServerError)
			return
		}

		path, err := s.store.GetFolderPath(id)
		if err != nil {
			log.Printf("failed to retrieve folder path %v: %v", id, err)
			http.Error(w, fmt.Sprintf("Failed to retrieve folder: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, folderToResponse(folder, path))
	}
}

func (s Server) entryFolderPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseFileFolderMoveRequest(r)
		if err != nil {
			log.Printf("invalid file move request: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		err = s.store.SetFileFolder(req.EntryID, req.FolderID)
		if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		} else if _, ok := errors.AsType[store.FolderNotFoundError](err); ok {
			http.Error(w, "Destination folder not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("failed to move file %v: %v", req.EntryID, err)
			http.Error(w, fmt.Sprintf("Failed to move file: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func foldersToResponse(folders []picoshare.Folder) []map[string]any {
	response := make([]map[string]any, 0, len(folders))
	for _, folder := range folders {
		var parentID *string
		if folder.ParentID != nil {
			s := folder.ParentID.String()
			parentID = &s
		}
		response = append(response, map[string]any{
			"id":       folder.ID.String(),
			"name":     folder.Name.String(),
			"parentId": parentID,
		})
	}

	return response
}

func folderToResponse(folder picoshare.Folder, path []picoshare.Folder) map[string]any {
	var parentID *string
	if folder.ParentID != nil {
		s := folder.ParentID.String()
		parentID = &s
	}

	pathResponse := make([]map[string]any, 0, len(path))
	for _, segment := range path {
		pathResponse = append(pathResponse, map[string]any{
			"id":   segment.ID.String(),
			"name": segment.Name.String(),
		})
	}

	return map[string]any{
		"id":       folder.ID.String(),
		"name":     folder.Name.String(),
		"parentId": parentID,
		"path":     pathResponse,
	}
}
