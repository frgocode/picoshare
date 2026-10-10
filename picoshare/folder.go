package picoshare

import (
	"fmt"
	"time"

	"github.com/mtlynch/picoshare/random"
)

// FolderIDLength is the number of characters in a folder ID.
const FolderIDLength = 10

// folderIDCharacters omits visually similar characters (I, l, 1), (0, O).
const folderIDCharacters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var folderIDCharacterSet = func() map[rune]struct{} {
	characters := make(map[rune]struct{}, len(folderIDCharacters))
	for _, character := range folderIDCharacters {
		characters[character] = struct{}{}
	}
	return characters
}()

// FolderID identifies a virtual folder.
type FolderID struct {
	value string
}

func (id FolderID) String() string {
	return id.value
}

// Empty returns true when the folder ID is empty (used for root references).
func (id FolderID) Empty() bool {
	return id.value == ""
}

// Equal returns true when two folder IDs are equal.
func (id FolderID) Equal(other FolderID) bool {
	return id.value == other.value
}

// FolderIDFromString constructs a folder ID from user-provided text.
func FolderIDFromString(raw string) (FolderID, error) {
	if len(raw) != FolderIDLength {
		return FolderID{}, fmt.Errorf(
			"folder ID has invalid length: got %d, want %d", len(raw), FolderIDLength)
	}

	for _, character := range raw {
		if _, ok := folderIDCharacterSet[character]; !ok {
			return FolderID{}, fmt.Errorf("folder ID contains invalid character: %q", character)
		}
	}

	return FolderID{value: raw}, nil
}

// NewFolderID generates a folder ID.
func NewFolderID() FolderID {
	return FolderID{value: random.String(FolderIDLength, []rune(folderIDCharacters))}
}

// MustCreateFolderID constructs a folder ID or panics when raw is invalid.
func MustCreateFolderID(raw string) FolderID {
	id, err := FolderIDFromString(raw)
	if err != nil {
		panic(fmt.Sprintf("failed to create folder ID: %v", err))
	}
	return id
}

// FolderName is a validated virtual folder name.
type FolderName string

func (n FolderName) String() string {
	return string(n)
}

// Folder is a virtual folder for logical file organization.
type Folder struct {
	ID       FolderID
	Name     FolderName
	ParentID *FolderID
	Created  time.Time
}
