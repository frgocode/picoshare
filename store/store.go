package store

import (
	"fmt"

	"github.com/mtlynch/picoshare/picoshare"
)

// EntryNotFoundError occurs when no entry exists with the given ID.
type EntryNotFoundError struct {
	ID picoshare.EntryID
}

func (f EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entry with ID %v", f.ID)
}

// GuestLinkNotFoundError occurs when no guest link exists with the given ID.
type GuestLinkNotFoundError struct {
	ID picoshare.GuestLinkID
}

func (f GuestLinkNotFoundError) Error() string {
	return fmt.Sprintf("Could not find guest link with ID %v", f.ID)
}

// FolderNotFoundError occurs when no folder exists with the given ID.
type FolderNotFoundError struct {
	ID picoshare.FolderID
}

func (f FolderNotFoundError) Error() string {
	return fmt.Sprintf("Could not find folder with ID %v", f.ID)
}

// DuplicateFolderNameError occurs when a folder name already exists among
// siblings.
type DuplicateFolderNameError struct {
	Name FolderNameDescription
}

// FolderNameDescription describes a folder name conflict for error messages.
type FolderNameDescription struct {
	Name string
}

func (f DuplicateFolderNameError) Error() string {
	return fmt.Sprintf("A folder named %q already exists in this location", f.Name.Name)
}

// FolderNotEmptyError occurs when deleting a folder that still contains
// subfolders or files.
type FolderNotEmptyError struct {
	ID picoshare.FolderID
}

func (f FolderNotEmptyError) Error() string {
	return fmt.Sprintf("Folder with ID %v is not empty", f.ID)
}

// InvalidFolderParentError occurs when a folder parent reference is invalid.
type InvalidFolderParentError struct {
	ID picoshare.FolderID
}

func (f InvalidFolderParentError) Error() string {
	return fmt.Sprintf("Invalid parent folder with ID %v", f.ID)
}

// FolderCycleError occurs when a folder move would create a circular
// hierarchy.
type FolderCycleError struct {
	ID picoshare.FolderID
}

func (f FolderCycleError) Error() string {
	return fmt.Sprintf("Moving folder with ID %v would create a circular hierarchy", f.ID)
}
