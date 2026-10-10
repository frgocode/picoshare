package parse

import (
	"errors"
	"strings"

	"github.com/mtlynch/picoshare/picoshare"
)

// MaxFolderNameBytes is the maximum number of bytes allowed for folder names.
const MaxFolderNameBytes = 255

var ErrFolderNameEmpty = errors.New("folder name must be non-empty")
var ErrFolderNameTooLong = errors.New("folder name too long")
var ErrFolderNameHasDotPrefix = errors.New("folder name cannot begin with dots")
var ErrFolderNameIllegalCharacters = errors.New("illegal characters in folder name")

func FolderName(s string) (picoshare.FolderName, error) {
	if s == "" {
		return picoshare.FolderName(""), ErrFolderNameEmpty
	}
	if len(s) > MaxFolderNameBytes {
		return picoshare.FolderName(""), ErrFolderNameTooLong
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return picoshare.FolderName(""), ErrFolderNameHasDotPrefix
	}
	if strings.ContainsAny(s, "\\/\a\b\t\n\v\f\r\n") {
		return picoshare.FolderName(""), ErrFolderNameIllegalCharacters
	}
	return picoshare.FolderName(s), nil
}
