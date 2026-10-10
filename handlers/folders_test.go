package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/store/test_sqlite"
)

func TestFolderPost(t *testing.T) {
	for _, tt := range []struct {
		explanation  string
		name         string
		parentID     string
		status       int
		setupFolders int
	}{
		{
			explanation: "creates a folder at root",
			name:        "Projects",
			status:      http.StatusOK,
		},
		{
			explanation: "rejects empty folder name",
			name:        "",
			status:      http.StatusBadRequest,
		},
		{
			explanation: "rejects folder name with slashes",
			name:        "a/b",
			status:      http.StatusBadRequest,
		},
		{
			explanation: "rejects folder name that is just a dot",
			name:        ".",
			status:      http.StatusBadRequest,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			dataStore := test_sqlite.New(t)
			s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

			payload, _ := json.Marshal(map[string]string{"name": tt.name})
			if tt.parentID != "" {
				payload, _ = json.Marshal(map[string]string{"name": tt.name, "parentId": tt.parentID})
			}
			req := httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader(payload))
			req.Header.Add("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			s.Router().ServeHTTP(rec, req)
			res := rec.Result()

			if got, want := res.StatusCode, tt.status; got != want {
				t.Fatalf("status=%d, want=%d", got, want)
			}
		})
	}
}

func TestFolderDuplicateName(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	createFolderViaAPI := func(name string) int {
		payload, _ := json.Marshal(map[string]string{"name": name})
		req := httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader(payload))
		req.Header.Add("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	if got, want := createFolderViaAPI("Docs"), http.StatusOK; got != want {
		t.Fatalf("first create status=%d, want=%d", got, want)
	}
	if got, want := createFolderViaAPI("Docs"), http.StatusConflict; got != want {
		t.Fatalf("duplicate create status=%d, want=%d", got, want)
	}
}

func TestFolderRenameAndDelete(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	folderID := createFolderHelper(t, s, "Original", "")

	renamePayload, _ := json.Marshal(map[string]string{"name": "Renamed"})
	{
		req := httptest.NewRequest(http.MethodPut, "/api/folders/"+folderID, bytes.NewReader(renamePayload))
		req.Header.Add("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
			t.Fatalf("rename status=%d, want=%d", got, want)
		}
	}

	{
		req := httptest.NewRequest(http.MethodDelete, "/api/folders/"+folderID, nil)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
			t.Fatalf("delete status=%d, want=%d", got, want)
		}
	}
}

func TestFolderDeleteNonemptyIsRejected(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	parentID := createFolderHelper(t, s, "Parent", "")
	createFolderHelper(t, s, "Child", parentID)

	req := httptest.NewRequest(http.MethodDelete, "/api/folders/"+parentID, nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusConflict; got != want {
		t.Fatalf("delete nonempty status=%d, want=%d", got, want)
	}
}

func TestFolderMoveCycleIsRejected(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	parentID := createFolderHelper(t, s, "Parent", "")
	childID := createFolderHelper(t, s, "Child", parentID)

	payload, _ := json.Marshal(map[string]any{"parentId": childID})
	req := httptest.NewRequest(http.MethodPut, "/api/folders/"+parentID+"/move", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("cycle move status=%d, want=%d", got, want)
	}
}

func TestFolderMoveToRoot(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	parentID := createFolderHelper(t, s, "Parent", "")
	childID := createFolderHelper(t, s, "Child", parentID)

	payload, _ := json.Marshal(map[string]any{"parentId": nil})
	req := httptest.NewRequest(http.MethodPut, "/api/folders/"+childID+"/move", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
		t.Fatalf("move to root status=%d, want=%d", got, want)
	}
}

func TestFileMoveBetweenFolders(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	folderA := createFolderHelper(t, s, "FolderA", "")
	folderB := createFolderHelper(t, s, "FolderB", "")

	entryID := uploadFileHelper(t, s, "movable.txt", "movable contents", "")

	movePayload, _ := json.Marshal(map[string]any{"folderId": folderA})
	{
		req := httptest.NewRequest(http.MethodPut, "/api/entry/"+entryID+"/folder", bytes.NewReader(movePayload))
		req.Header.Add("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
			t.Fatalf("move to A status=%d, want=%d", got, want)
		}
	}

	movePayload, _ = json.Marshal(map[string]any{"folderId": folderB})
	{
		req := httptest.NewRequest(http.MethodPut, "/api/entry/"+entryID+"/folder", bytes.NewReader(movePayload))
		req.Header.Add("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
			t.Fatalf("move to B status=%d, want=%d", got, want)
		}
	}

	movePayload, _ = json.Marshal(map[string]any{"folderId": nil})
	{
		req := httptest.NewRequest(http.MethodPut, "/api/entry/"+entryID+"/folder", bytes.NewReader(movePayload))
		req.Header.Add("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		if got, want := rec.Result().StatusCode, http.StatusOK; got != want {
			t.Fatalf("move to root status=%d, want=%d", got, want)
		}
	}

	folderID, err := picoshare.FolderIDFromString(folderA)
	if err != nil {
		t.Fatalf("failed to parse folder ID: %v", err)
	}
	filesInA, err := dataStore.GetEntriesMetadataInFolder(&folderID)
	if err != nil {
		t.Fatalf("failed to list folder A: %v", err)
	}
	if got, want := len(filesInA), 0; got != want {
		t.Fatalf("folder A should be empty after moves, count=%d, want=%d", got, want)
	}
}

func TestUploadIntoFolder(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	folderID := createFolderHelper(t, s, "UploadTarget", "")

	entryID := uploadFileHelper(t, s, "into-folder.txt", "folder contents", folderID)

	parsedFolder, err := picoshare.FolderIDFromString(folderID)
	if err != nil {
		t.Fatalf("failed to parse folder ID: %v", err)
	}
	files, err := dataStore.GetEntriesMetadataInFolder(&parsedFolder)
	if err != nil {
		t.Fatalf("failed to list folder: %v", err)
	}
	if got, want := len(files), 1; got != want {
		t.Fatalf("folder file count=%d, want=%d", got, want)
	}

	rootFiles, err := dataStore.GetEntriesMetadataInFolder(nil)
	if err != nil {
		t.Fatalf("failed to list root: %v", err)
	}
	if got, want := len(rootFiles), 0; got != want {
		t.Fatalf("root should be empty, count=%d, want=%d", got, want)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/-"+entryID, nil)
	downloadRec := httptest.NewRecorder()
	s.Router().ServeHTTP(downloadRec, downloadReq)
	downloadRes := downloadRec.Result()
	if got, want := downloadRes.StatusCode, http.StatusOK; got != want {
		t.Fatalf("download status=%d, want=%d", got, want)
	}
	body, _ := io.ReadAll(downloadRes.Body)
	if got, want := string(body), "folder contents"; got != want {
		t.Errorf("download contents=%q, want=%q", got, want)
	}
}

func TestFileMoveInvalidReferencesAreRejected(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	entryID := uploadFileHelper(t, s, "file.txt", "contents", "")

	bogusFolder := "AAAAAAAAAA"
	payload, _ := json.Marshal(map[string]any{"folderId": bogusFolder})
	req := httptest.NewRequest(http.MethodPut, "/api/entry/"+entryID+"/folder", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("move to missing folder status=%d, want=%d", got, want)
	}

	payload, _ = json.Marshal(map[string]any{"folderId": nil})
	req = httptest.NewRequest(http.MethodPut, "/api/entry/BBBBBBBBBB/folder", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("move missing file status=%d, want=%d", got, want)
	}
}

func TestFolderAPIsRequireValidIDs(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	req := httptest.NewRequest(http.MethodGet, "/api/folders/invalid", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got, want := rec.Result().StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("invalid folder ID status=%d, want=%d", got, want)
	}

	payload, _ := json.Marshal(map[string]any{"parentId": "not-a-folder"})
	req = httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if got := rec.Result().StatusCode; got != http.StatusBadRequest {
		t.Fatalf("invalid parent status=%d, want=%d", got, http.StatusBadRequest)
	}
}

func TestFileIndexShowsFoldersAndBreadcrumbs(t *testing.T) {
	dataStore := test_sqlite.New(t)
	s := handlers.New(mockAuthenticator{}, &dataStore, nilSpaceCheckFunc, nilGarbageCollector, time.Now)

	parentID := createFolderHelper(t, s, "ParentNav", "")
	childID := createFolderHelper(t, s, "ChildNav", parentID)
	uploadFileHelper(t, s, "root-file.txt", "root", "")
	uploadFileHelper(t, s, "child-file.txt", "child", childID)

	{
		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		res := rec.Result()
		if got, want := res.StatusCode, http.StatusOK; got != want {
			t.Fatalf("root files status=%d, want=%d", got, want)
		}
		body, _ := io.ReadAll(res.Body)
		if !strings.Contains(string(body), "ParentNav") {
			t.Errorf("root page should list ParentNav folder")
		}
		if !strings.Contains(string(body), "root-file.txt") {
			t.Errorf("root page should list root-file.txt")
		}
		if strings.Contains(string(body), "child-file.txt") {
			t.Errorf("root page should not list child-file.txt")
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/files?folder="+childID, nil)
		rec := httptest.NewRecorder()
		s.Router().ServeHTTP(rec, req)
		res := rec.Result()
		if got, want := res.StatusCode, http.StatusOK; got != want {
			t.Fatalf("child files status=%d, want=%d", got, want)
		}
		body, _ := io.ReadAll(res.Body)
		if !strings.Contains(string(body), "child-file.txt") {
			t.Errorf("child page should list child-file.txt")
		}
		if !strings.Contains(string(body), "ParentNav") {
			t.Errorf("child page should show breadcrumb with ParentNav")
		}
	}
}

func createFolderHelper(t *testing.T, s handlers.Server, name, parentID string) string {
	t.Helper()
	payloadMap := map[string]string{"name": name}
	if parentID != "" {
		payloadMap["parentId"] = parentID
	}
	payload, _ := json.Marshal(payloadMap)
	req := httptest.NewRequest(http.MethodPost, "/api/folders", bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("failed to create folder %q: status=%d body=%s", name, res.StatusCode, string(body))
	}
	var response handlers.FolderPostResponse
	body, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("failed to parse folder create response: %v", err)
	}
	return response.ID
}

func uploadFileHelper(t *testing.T, s handlers.Server, filename, contents, folderID string) string {
	t.Helper()
	url := "/api/entry?expiration=2040-01-01T00%3A00%3A00Z"
	if folderID != "" {
		url += "&folder=" + folderID
	}
	formData, contentType := createMultipartFormBody(filename, "", "", strings.NewReader(contents))
	req := httptest.NewRequest(http.MethodPost, url, formData)
	req.Header.Add("Content-Type", contentType)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("failed to upload %q: status=%d body=%s", filename, res.StatusCode, string(body))
	}
	var response handlers.EntryPostResponse
	body, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("failed to parse upload response: %v", err)
	}
	return response.ID
}
