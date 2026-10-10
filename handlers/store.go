package handlers

import (
	"io"

	"github.com/mtlynch/picoshare/picoshare"
)

type Store interface {
	GetEntriesMetadata() ([]picoshare.UploadMetadata, error)
	ReadEntryFile(picoshare.EntryID) (io.ReadSeeker, error)
	GetEntryMetadata(id picoshare.EntryID) (picoshare.UploadMetadata, error)
	InsertEntry(reader io.Reader, metadata picoshare.UploadMetadata) error
	UpdateEntryMetadata(id picoshare.EntryID, metadata picoshare.UploadMetadata) error
	DeleteEntry(id picoshare.EntryID) error
	GetGuestLink(picoshare.GuestLinkID) (picoshare.GuestLink, error)
	GetGuestLinks() ([]picoshare.GuestLink, error)
	InsertGuestLink(picoshare.GuestLink) error
	DeleteGuestLink(picoshare.GuestLinkID) error
	DisableGuestLink(picoshare.GuestLinkID) error
	EnableGuestLink(picoshare.GuestLinkID) error
	InsertEntryDownload(picoshare.EntryID, picoshare.DownloadRecord) error
	GetEntryDownloads(id picoshare.EntryID) ([]picoshare.DownloadRecord, error)
	ReadSettings() (picoshare.Settings, error)
	UpdateSettings(picoshare.Settings) error
	CreateFolder(name picoshare.FolderName, parentID *picoshare.FolderID) (picoshare.Folder, error)
	GetFolder(id picoshare.FolderID) (picoshare.Folder, error)
	ListChildFolders(parentID *picoshare.FolderID) ([]picoshare.Folder, error)
	ListAllFolders() ([]picoshare.Folder, error)
	GetFolderPath(id picoshare.FolderID) ([]picoshare.Folder, error)
	RenameFolder(id picoshare.FolderID, name picoshare.FolderName) error
	MoveFolder(id picoshare.FolderID, newParentID *picoshare.FolderID) error
	DeleteFolder(id picoshare.FolderID) error
	SetFileFolder(entryID picoshare.EntryID, folderID *picoshare.FolderID) error
	GetFileFolder(entryID picoshare.EntryID) (*picoshare.Folder, error)
	GetEntriesMetadataInFolder(folderID *picoshare.FolderID) ([]picoshare.UploadMetadata, error)
}
