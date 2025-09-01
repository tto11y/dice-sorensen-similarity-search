package markdowndoc_test

import (
	"bytes"
	"context"
	"dice-sorensen-similarity-search/internal/database"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/markdowndoc"
	"dice-sorensen-similarity-search/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// ####################### valid behavior tests
func TestGetNavigationItemsTrees_Success(t *testing.T) {

	ctrl := newMockController(newMockRepository())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/markdown-doc/navigation-items", nil)
	c.Request = req

	ctrl.GetNavigationItemsTrees(c)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", w.Code)
		return
	}

	// unmarshal response
	var got []*markdowndoc.NavigationItem
	err := json.Unmarshal(w.Body.Bytes(), &got)
	if err != nil {
		t.Fatalf("unmarshalling error: %v", err)
	}

	// prevent UUID comparison
	setUuidNil(got)

	want := getExpectedNavigationItemTree()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}

	for _, v := range got {
		if v.Href == "Empty_Folder" {
			t.Errorf("got href Empty_Folder, want it to be removed")
			return
		}
	}
}

func newMockController(repo database.Repository) *markdowndoc.Controller {
	env := environment.Null()
	env.Repository = repo

	return &markdowndoc.Controller{
		Env: env,
		NavigationItemTreeService: markdowndoc.NavigationItemTreeService{
			Env:      env,
			Collator: collate.New(language.English),
		},
		MarkdownSearchMatchMapper: markdowndoc.MarkdownSearchMatchMapper{Env: env},
	}
}

func newMockRepository() *mockRepository {

	sampleMarkdownsWithPrefixedPath := []models.MarkdownContent{
		{
			Meta: models.MarkdownMeta{
				Path: "markdowns/001_intro",
				Name: "intro",
			},
			Content: "this is a another sample markdown content",
		},
	}

	samplePrefixedTopLevelMarkdowns := []models.MarkdownContent{
		{
			Meta: models.MarkdownMeta{
				Path: "markdowns",
				Name: "1_prefixed_top-level_Markdown",
			},
			Content: "this is a prefixed top-level markdown content",
		},
	}

	sampleMarkdownsForSearch := []models.MarkdownContent{
		{
			Meta:    models.MarkdownMeta{Name: "example", Path: "markdowns/example"},
			Content: "this is a sample markdown content",
		},
		{
			Meta:    models.MarkdownMeta{Name: "hidden", Path: "markdowns/.hidden"},
			Content: "this is a hidden markdown content",
		},
		{
			Meta:    models.MarkdownMeta{Name: "1_prefix-removed", Path: "markdowns/prefixed"},
			Content: "this is a sample markdown content with prefix root element",
		},
		{
			Meta:    models.MarkdownMeta{Name: "unmatched", Path: "markdowns/testy"},
			Content: "does NOT match",
		},
		{
			Meta:    models.MarkdownMeta{Name: "second_match", Path: "markdowns/example"},
			Content: "this is a another sample markdown content",
		},
		{
			Meta: models.MarkdownMeta{
				Name: "something_else",
				Path: "123_root/section",
			},
			Content: "this is a another sample markdown content",
		},
		{
			Meta: models.MarkdownMeta{
				Path: "root/section",
				Name: "another",
			},
			Content: "this is a another sample markdown content",
		},
		{
			Meta: models.MarkdownMeta{
				Path: "123-root/7_section",
				Name: "seven",
			},
			Content: "this is a another sample markdown content",
		},
		{
			Meta: models.MarkdownMeta{
				Path: "123.root/2_section",
				Name: "numbered",
			},
			Content: "this is a another sample markdown content",
		},
	}

	sampleMarkdownsForSearch = append(sampleMarkdownsForSearch, sampleMarkdownsWithPrefixedPath...)
	sampleMarkdownsForSearch = append(sampleMarkdownsForSearch, samplePrefixedTopLevelMarkdowns...)

	return &mockRepository{
		markdownMetas: []models.MarkdownMeta{
			{Name: "1_Onboarding", Path: "markdowns/Gateway", CharCount: 350},
			{Name: "2_Data_Preparation", Path: "markdowns/Gateway", CharCount: 350},
			{Name: "3_Visualization", Path: "markdowns/Gateway", CharCount: 350},
			{Name: "4_Technical_Docs", Path: "markdowns/Gateway", CharCount: 350},
			{Name: "5_OpenTelemetry", Path: "markdowns/Gateway", CharCount: 350},
			{Name: "Getting_Started", Path: "markdowns/Guidelines", CharCount: 350},
			{Name: "Migration_Guide", Path: "markdowns/Guidelines", CharCount: 350},
			{Name: "Empty1", Path: "markdowns/Guidelines", CharCount: 0},
			{Name: "Empty2", Path: "markdowns/Guidelines", CharCount: 0},
			{Name: "Empty_Markdown", Path: "markdowns/Empty_Folder", CharCount: 0},
		},
		markdownContentsForSearch: sampleMarkdownsForSearch,
		markdownsWithPrefixedPath: sampleMarkdownsWithPrefixedPath,
		prefixedTopLevelMarkdowns: samplePrefixedTopLevelMarkdowns,
	}
}

func getExpectedNavigationItemTree() []*markdowndoc.NavigationItem {
	want := []*markdowndoc.NavigationItem{
		{
			Label:    "Gateway",
			Href:     "Gateway",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "Guidelines",
			Href:     "Guidelines",
			Parent:   nil,
			Children: nil,
		},
	}

	wantGatewayChildren := []*markdowndoc.NavigationItem{
		{
			Label:    "1 Onboarding",
			Href:     "1_Onboarding",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "2 Data Preparation",
			Href:     "2_Data_Preparation",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "3 Visualization",
			Href:     "3_Visualization",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "4 Technical Docs",
			Href:     "4_Technical_Docs",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "5 OpenTelemetry",
			Href:     "5_OpenTelemetry",
			Parent:   nil,
			Children: nil,
		},
	}
	wantGuidelinesChildren := []*markdowndoc.NavigationItem{
		{
			Label:    "Getting Started",
			Href:     "Getting_Started",
			Parent:   nil,
			Children: nil,
		},
		{
			Label:    "Migration Guide",
			Href:     "Migration_Guide",
			Parent:   nil,
			Children: nil,
		},
	}

	want[0].Children = wantGatewayChildren
	want[1].Children = wantGuidelinesChildren

	return want
}

func setUuidNil(items []*markdowndoc.NavigationItem) {
	if len(items) == 0 {
		return
	}

	for i := 0; i < len(items); i++ {
		items[i].Uuid = ""

		if items[i].Children != nil {
			setUuidNil(items[i].Children)
		}
	}
}

func TestGetMarkdownByName_Success(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{{Key: "name", Value: "guide"}}

	mock := &mockRepository{
		markdownContent: map[string]models.MarkdownContent{
			"guide": {Content: "# Welcome"},
		},
	}

	ctrl := newMockController(mock)

	req := httptest.NewRequest(http.MethodGet, "/markdown-doc/markdown/guide", nil)
	c.Request = req

	ctrl.GetMarkdownByName(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
		return
	}

	if !strings.Contains(w.Body.String(), "Welcome") {
		t.Errorf("expected response body to contain content")
		return
	}
}

func TestGetMarkdownSearchTermMatches_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockedRepo := newMockRepository()
	ctrl := newMockController(mockedRepo)

	wantTerm := "this"
	wantPageSize := 13
	wantNumberOfMatches := 9

	payload := markdowndoc.MarkdownSearchPayload{
		Term: wantTerm,
		Pageable: markdowndoc.Pageable{
			PageSize:   wantPageSize,
			PageNumber: 1,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", w.Code)
		return
	}

	var page markdowndoc.Page[markdowndoc.MarkdownSearchMatch]
	err = json.Unmarshal(w.Body.Bytes(), &page)
	if err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
		return
	}

	if len(page.Content) != wantNumberOfMatches {
		t.Errorf("want %d matches, got %d", wantNumberOfMatches, len(page.Content))
		return
	}

	compiledRegex, err := regexp.Compile("^\\d+")
	if err != nil {
		t.Errorf("failed to compile regex: %v", err)
		return
	}

	for _, match := range page.Content {
		if strings.HasPrefix(match.Path, ".") {
			t.Errorf("want 0 hidden elements, got label \"%s\" / path \"%s\"", match.Label, match.Path)
			return
		}

		if compiledRegex.MatchString(match.Path) {
			t.Errorf("want 0 prefixed path roots, got label \"%s\" / path \"%s\"", match.Label, match.Path)
			return
		}

		if compiledRegex.MatchString(match.PrettyPath) {
			t.Errorf("want 0 prefixed pretty path roots, got pretty path \"%s\" / path \"%s\"", match.PrettyPath, match.Path)
			return
		}

		if len(match.Path) > 0 {
			unprettifiedPath := strings.ReplaceAll(match.PrettyPath, " ", "_")
			if unprettifiedPath != match.Path {
				t.Errorf("want path and pretty path to the same (except for the separator), got pretty path \"%s\" / unprettified path \"%s\" / path \"%s\"", match.PrettyPath, unprettifiedPath, match.Path)
				return
			}
		}

		// check if the Label and Href of top-level Markdowns that are prefixed and contain underscores are properly handled
		if len(match.Path) == 0 { // top-level Markdown
			wantName := mockedRepo.prefixedTopLevelMarkdowns[0].Meta.Name
			prefix := compiledRegex.FindString(wantName)
			unprefixedName := strings.TrimPrefix(wantName, prefix+"_")
			wantPrettifiedUnprefixedName := strings.ReplaceAll(unprefixedName, "_", " ")

			if match.Label != wantPrettifiedUnprefixedName {
				t.Errorf("want top-level Markdown label without prefix (%s), got label=%s / path=%s", wantPrettifiedUnprefixedName, match.Label, match.Path)
				return
			}

			if match.Href != wantName {
				t.Errorf("want top-level Markdown Href with prefix (%s), got href=%s / path=%s", wantName, match.Href, match.Path)
				return
			}
		}

		if match.MatchingText != wantTerm {
			t.Errorf("want matching text %s, got %s", wantTerm, match.MatchingText)
			return
		}
	}

	wantTotalElements := 100
	if page.TotalElements != wantTotalElements {
		t.Errorf("want %d total elements, got %d", wantTotalElements, page.TotalElements)
		return
	}

	wantTotalPages := int(math.Ceil(float64(wantTotalElements) / float64(wantPageSize)))
	if page.TotalPages != wantTotalPages {
		t.Errorf("want %d total pages, got %d", wantTotalPages, page.TotalPages)
		return
	}
}

func TestGetMarkdownSearchTermMatches_Success_matchesOnlyHiddenMarkdowns(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrl := newMockController(newMockRepository())

	wantTerm := "hidden"
	wantPageSize := 13
	var wantTotalElements, wantNumberOfMatches int

	payload := markdowndoc.MarkdownSearchPayload{
		Term: wantTerm,
		Pageable: markdowndoc.Pageable{
			PageSize:   wantPageSize,
			PageNumber: 1,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", w.Code)
		return
	}

	var page markdowndoc.Page[markdowndoc.MarkdownSearchMatch]
	err = json.Unmarshal(w.Body.Bytes(), &page)
	if err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
		return
	}

	if len(page.Content) != wantNumberOfMatches {
		t.Errorf("want %d matches, got %d", wantNumberOfMatches, len(page.Content))
		return
	}

	if page.TotalElements != wantTotalElements {
		t.Errorf("want %d total elements, got %d", wantTotalElements, page.TotalElements)
		return
	}

	wantTotalPages := int(math.Ceil(float64(wantTotalElements) / float64(wantPageSize)))
	if page.TotalPages != wantTotalPages {
		t.Errorf("want %d total pages, got %d", wantTotalPages, page.TotalPages)
		return
	}
}

// ####################### invalid behavior tests
func TestGetNavigationItemsTrees_DBError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	mock := &mockRepository{findMetasErr: errors.New("DB unreachable")}
	ctrl := newMockController(mock)

	req := httptest.NewRequest(http.MethodGet, "/markdown-doc/navigation-items", nil)
	c.Request = req

	ctrl.GetNavigationItemsTrees(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
		return
	}
}

func TestGetMarkdownByName_MissingParam(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ctrl := newMockController(&mockRepository{})

	req := httptest.NewRequest(http.MethodGet, "/markdown-doc/markdown/", nil)
	c.Request = req

	ctrl.GetMarkdownByName(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
		return
	}
}

func TestGetMarkdownByName_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{{Key: "name", Value: "ghost"}}

	mock := &mockRepository{
		markdownContent: map[string]models.MarkdownContent{},
	}

	ctrl := newMockController(mock)

	req := httptest.NewRequest(http.MethodGet, "/markdown-doc/markdown/ghost", nil)
	c.Request = req

	ctrl.GetMarkdownByName(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
		return
	}
}

func TestGetMarkdownSearchTermMatches_ErrorDuringFind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockedRepo := newMockRepository()
	mockedRepo.findMarkdownsBySearchTermSimpleErr = errors.New("could not find markdown by search term")
	ctrl := newMockController(mockedRepo)

	wantTerm := "this"
	wantPageSize := 9

	payload := markdowndoc.MarkdownSearchPayload{
		Term: wantTerm,
		Pageable: markdowndoc.Pageable{
			PageSize:   wantPageSize,
			PageNumber: 1,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "error reading Markdown search matches") {
		t.Errorf("want 'error reading Markdown search matches', got %s", w.Body.String())
		return
	}
}

func TestGetMarkdownSearchTermMatches_ErrorDuringCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockedRepo := newMockRepository()
	mockedRepo.countMarkdownsMatchesBySearchTermSimpleErr = errors.New("could not count matches for search term")
	ctrl := newMockController(mockedRepo)

	wantTerm := "this"
	wantPageSize := 9

	payload := markdowndoc.MarkdownSearchPayload{
		Term: wantTerm,
		Pageable: markdowndoc.Pageable{
			PageSize:   wantPageSize,
			PageNumber: 1,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "error counting Markdown search matches") {
		t.Errorf("want 'error counting Markdown search matches', got %s", w.Body.String())
		return
	}
}

func TestGetMarkdownSearchTermMatches_BadRequestBecauseNoSearchTerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockedRepo := newMockRepository()
	ctrl := newMockController(mockedRepo)

	wantPageSize := 9

	payload := markdowndoc.MarkdownSearchPayload{
		Pageable: markdowndoc.Pageable{
			PageSize:   wantPageSize,
			PageNumber: 1,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "did not perform search because no search term was present") {
		t.Errorf("want 'did not perform search because no search term was present', got %s", w.Body.String())
		return
	}
}

func TestGetMarkdownSearchTermMatches_BadRequestBecauseInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockedRepo := newMockRepository()
	ctrl := newMockController(mockedRepo)

	// Intentionally malformed JSON (missing closing brace)
	invalidJSON := `{"term": "test", "pageable": {"pageSize": 5`

	req, err := http.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(invalidJSON))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.GetMarkdownSearchTermMatches(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "error while unmarshaling request body") {
		t.Errorf("want error message about unmarshaling, got %s", w.Body.String())
		return
	}
}

// ####################### creating mocks
type mockRepository struct {
	findMetasErr    error
	markdownMetas   []models.MarkdownMeta
	markdownContent map[string]models.MarkdownContent
	// the slice below contains also markdownsWithPrefixedPath and prefixedTopLevelMarkdowns
	markdownContentsForSearch                  []models.MarkdownContent
	markdownsWithPrefixedPath                  []models.MarkdownContent
	prefixedTopLevelMarkdowns                  []models.MarkdownContent
	findMarkdownsBySearchTermSimpleErr         error
	countMarkdownsMatchesBySearchTermSimpleErr error
}

func (m *mockRepository) FindMarkdownsBySearchTermSimple(ctx context.Context, term string, results *[]models.MarkdownContent) error {
	if m.findMarkdownsBySearchTermSimpleErr != nil {
		return m.findMarkdownsBySearchTermSimpleErr
	}

	for _, data := range m.markdownContentsForSearch {
		pathElements := strings.Split(data.Meta.Path, "/")

		if len(pathElements) < 2 {
			slog.Info("checking top-level Markdown file")
			if !strings.Contains(data.Content, term) || strings.HasPrefix(data.Meta.Name, ".") {
				continue
			}
		}

		if len(pathElements) >= 2 {
			slog.Info("checking top-level folder")
			if !strings.Contains(data.Content, term) || strings.HasPrefix(pathElements[1], ".") {
				continue
			}
		}

		*results = append(*results, data)
	}

	return nil
}

func (m *mockRepository) CountMarkdownsMatchesBySearchTermSimple(ctx context.Context, term string, count *int) error {
	if m.countMarkdownsMatchesBySearchTermSimpleErr != nil {
		return m.countMarkdownsMatchesBySearchTermSimpleErr
	}

	// simulating that matches only occur in hidden elements
	if term == "hidden" {
		*count = 0
		return nil
	}

	*count = 100
	return nil
}

func (m *mockRepository) FindAllMarkdownMetas(_ context.Context, metas *[]models.MarkdownMeta) error {
	if m.findMetasErr != nil {
		return m.findMetasErr
	}
	*metas = m.markdownMetas
	return nil
}

func (m *mockRepository) FindMarkdownMetasWhereCharCountGreaterThan(ctx context.Context, x int, markdownMetas *[]models.MarkdownMeta) error {
	if m.findMetasErr != nil {
		return m.findMetasErr
	}

	for _, v := range m.markdownMetas {
		if v.CharCount == 0 {
			continue
		}
		*markdownMetas = append(*markdownMetas, v)
	}

	return nil
}

func (m *mockRepository) FindMarkdownContentByName(_ context.Context, name string, content *models.MarkdownContent) error {
	c, ok := m.markdownContent[name]
	if !ok {
		return fmt.Errorf("not found")
	}
	*content = c
	return nil
}

func (m *mockRepository) DeleteMarkdownMetasByIds(_ context.Context, ids []uint) error {
	return nil
}

func (m *mockRepository) DeleteMarkdownContentsByIds(_ context.Context, ids []uint) error {
	return nil
}

func (m *mockRepository) FindUserLoginCredentials(_ context.Context, _ string, _ *models.User) error {
	return nil
}

func (m *mockRepository) FindMarkdownContentIdsByMetaIds(_ context.Context, _ []uint, out *[]uint) error {
	return nil
}

func (m *mockRepository) UpsertMarkdownMetas(_ context.Context, metas []models.MarkdownMeta) error {
	return nil
}

func (m *mockRepository) UpsertMarkdownContents(_ context.Context, _ []models.MarkdownContent) error {
	return nil
}
