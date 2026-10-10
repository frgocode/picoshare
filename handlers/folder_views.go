package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store"
)

func (s Server) folderEditGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/folder-edit.html")

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
			log.Printf("error retrieving folder %v: %v", id, err)
			http.Error(w, "Failed to retrieve folder", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			Folder picoshare.Folder
		}{
			commonProps: makeCommonProps("PicoShare - Rename Folder", r.Context()),
			Folder:      folder,
		})
	}
}

func (s Server) folderConfirmDeleteGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/folder-delete.html")

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
			log.Printf("error retrieving folder %v: %v", id, err)
			http.Error(w, "Failed to retrieve folder", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			Folder picoshare.Folder
		}{
			commonProps: makeCommonProps("PicoShare - Delete Folder", r.Context()),
			Folder:      folder,
		})
	}
}

func (s Server) folderMoveGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/folder-move.html")

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
			log.Printf("error retrieving folder %v: %v", id, err)
			http.Error(w, "Failed to retrieve folder", http.StatusInternalServerError)
			return
		}

		folders, err := s.store.ListAllFolders()
		if err != nil {
			log.Printf("failed to list folders: %v", err)
			http.Error(w, "Failed to retrieve folders", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			Folder  picoshare.Folder
			Folders []picoshare.Folder
		}{
			commonProps: makeCommonProps("PicoShare - Move Folder", r.Context()),
			Folder:      folder,
			Folders:     folders,
		})
	}
}

func (s Server) fileMoveGet() http.HandlerFunc {
	t := parseTemplates("templates/pages/file-move.html")

	return func(w http.ResponseWriter, r *http.Request) {
		id, err := picoshare.EntryIDFromString(mux.Vars(r)["id"])
		if err != nil {
			log.Printf("error parsing entry ID: %v", err)
			http.Error(w, fmt.Sprintf("Invalid file ID: %v", err), http.StatusBadRequest)
			return
		}

		metadata, err := s.store.GetEntryMetadata(id)
		if _, ok := errors.AsType[store.EntryNotFoundError](err); ok {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("error retrieving entry %v: %v", id, err)
			http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
			return
		}

		currentFolder, err := s.store.GetFileFolder(id)
		if err != nil {
			log.Printf("error retrieving file folder %v: %v", id, err)
			http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
			return
		}

		folders, err := s.store.ListAllFolders()
		if err != nil {
			log.Printf("failed to list folders: %v", err)
			http.Error(w, "Failed to retrieve folders", http.StatusInternalServerError)
			return
		}

		renderTemplate(w, t, struct {
			commonProps
			Metadata      picoshare.UploadMetadata
			CurrentFolder *picoshare.Folder
			Folders       []picoshare.Folder
		}{
			commonProps:   makeCommonProps("PicoShare - Move File", r.Context()),
			Metadata:      metadata,
			CurrentFolder: currentFolder,
			Folders:       folders,
		})
	}
}
