package bitbucket_test

import (
	"context"
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/models"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ####################### valid cases
func TestFetchMarkdownsFromBitbucket_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var core zapcore.Core

	wantCharCountByName := map[string]uint{
		"guide_asd.asd": 7, // length: 7 (NOTE: name is sanitized)
		"empty":         0, // length: 0 => should be filtered when building navigationItemsTrees because charCount is 0
	}

	// deep copy and set to invalid value
	gotCharCountByName := make(map[string]uint, len(wantCharCountByName))
	for k := range wantCharCountByName {
		gotCharCountByName[k] = 100_000_000
	}

	mockedRepo := &mockRepository{
		sanitizedName:   "guide-asd-asd",
		charCountByName: gotCharCountByName,
	}
	mockCtrl := &bitbucket.Controller{
		Env: &environment.Env{
			Repository: mockedRepo,
			Logger:     &logging.DefaultLogger{Logger: zap.New(core).Sugar()},
		},
		BitbucketReader: &mockBitbucketReader{
			files: []string{
				"dir/guide asd.asd.md",
				"dir/empty.md",
			},
			readContent: map[string]string{
				"dir/guide asd.asd.md": "# Hello", // length: 7
				"dir/empty.md":         "",        // length: 0 => should be filtered
			},
		},
		MarkdownHousekeeper: &mockHousekeeper{},
		ProjectName:         "CIM",
		RepositoryName:      "o11y-self-service-content",
	}

	mockCtrl.FetchMarkdownsFromBitbucket(c)

	want := http.StatusNoContent
	got := w.Code
	if got != want {
		t.Errorf("status code mismatch: got %d, want %d", got, want)
		return
	}

	if !mockedRepo.upsertContentsCalled {
		t.Error("upsert contents was not called")
		return
	}

	if !mockedRepo.upsertMetasCalled {
		t.Error("upsert metas was not called")
		return
	}

	if !cmp.Equal(wantCharCountByName, gotCharCountByName) {
		t.Error(cmp.Diff(wantCharCountByName, gotCharCountByName))
		return
	}
}

// ####################### invalid cases
func TestFetchMarkdownsFromBitbucket_ReadStructureFails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var core zapcore.Core

	mockCtrl := &bitbucket.Controller{
		Env: &environment.Env{Logger: &logging.DefaultLogger{Logger: zap.New(core).Sugar()}},
		BitbucketReader: &mockBitbucketReader{
			failList: true,
		},
	}

	mockCtrl.FetchMarkdownsFromBitbucket(c)

	want := http.StatusInternalServerError
	got := w.Code
	if got != want {
		t.Errorf("status code mismatch: got %d, want %d", got, want)
		return
	}
}

func TestFetchMarkdownsFromBitbucket_DBMetaFetchFails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var core zapcore.Core

	mockCtrl := &bitbucket.Controller{
		Env: &environment.Env{
			Repository: &mockRepository{
				findErr:       errors.New("selecting markdown metas failed"),
				failMetaQuery: true,
			},
			Logger: &logging.DefaultLogger{Logger: zap.New(core).Sugar()},
		},
		BitbucketReader: &mockBitbucketReader{
			files:       []string{"doc/intro.md"},
			readContent: map[string]string{"doc/intro.md": "hi"},
		},
		MarkdownHousekeeper: &mockHousekeeper{},
	}

	mockCtrl.FetchMarkdownsFromBitbucket(c)

	want := http.StatusInternalServerError
	got := w.Code
	if got != want {
		t.Errorf("status code mismatch: got %d, want %d", got, want)
		return
	}
}

// ####################### creating mocks
type mockHousekeeper struct {
	called      bool
	inputBitMd  []models.MarkdownMeta
	inputDbMd   []models.MarkdownMeta
	returnError error
}

func (m *mockHousekeeper) DeleteObsoleteMarkdownsFromDatabase(
	ctx context.Context,
	markdownMetasFromBitbucket []models.MarkdownMeta,
	markdownMetasFromDb []models.MarkdownMeta,
) error {
	m.called = true
	m.inputBitMd = markdownMetasFromBitbucket
	m.inputDbMd = markdownMetasFromDb
	return m.returnError
}

type mockBitbucketReader struct {
	files       []string
	readContent map[string]string

	failList     bool
	failReadFile map[string]bool
}

func (m *mockBitbucketReader) ReadMarkdownFileStructureRecursively(projectName, repoName string, start, limit int) ([]string, error) {
	if m.failList {
		return nil, fmt.Errorf("failed to list markdown files")
	}
	return m.files, nil
}

func (m *mockBitbucketReader) ReadRepoRootFolderContent(projectName, repoName string) ([]string, error) {
	// Not needed for current tests, implement if required later
	return nil, nil
}

func (m *mockBitbucketReader) ReadFileContentAtRevision(projectName, repoName, filePath, revision string) (string, error) {
	if m.failReadFile != nil && m.failReadFile[filePath] {
		return "", fmt.Errorf("failed to read %s", filePath)
	}
	return m.readContent[filePath], nil
}
