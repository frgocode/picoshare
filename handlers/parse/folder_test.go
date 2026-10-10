package parse_test

import (
	"strings"
	"testing"

	"github.com/mtlynch/picoshare/handlers/parse"
	"github.com/mtlynch/picoshare/picoshare"
)

func TestFolderName(t *testing.T) {
	for _, tt := range []struct {
		explanation string
		input       string
		output      picoshare.FolderName
		err         error
	}{
		{
			explanation: "accepts a valid folder name",
			input:       "Projects",
			output:      picoshare.FolderName("Projects"),
			err:         nil,
		},
		{
			explanation: "accepts a folder name with spaces",
			input:       "My Projects",
			output:      picoshare.FolderName("My Projects"),
			err:         nil,
		},
		{
			explanation: "rejects an empty folder name",
			input:       "",
			err:         parse.ErrFolderNameEmpty,
		},
		{
			explanation: "rejects a folder name that is too long",
			input:       strings.Repeat("A", parse.MaxFolderNameBytes+1),
			err:         parse.ErrFolderNameTooLong,
		},
		{
			explanation: "rejects a folder name that is just a dot",
			input:       ".",
			err:         parse.ErrFolderNameHasDotPrefix,
		},
		{
			explanation: "rejects a folder name with forward slashes",
			input:       "a/b",
			err:         parse.ErrFolderNameIllegalCharacters,
		},
		{
			explanation: "rejects a folder name with backslashes",
			input:       `a\b`,
			err:         parse.ErrFolderNameIllegalCharacters,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			name, err := parse.FolderName(tt.input)
			if got, want := err, tt.err; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}
			if tt.err == nil {
				if got, want := name, tt.output; got != want {
					t.Errorf("name=%v, want=%v", got, want)
				}
			}
		})
	}
}
