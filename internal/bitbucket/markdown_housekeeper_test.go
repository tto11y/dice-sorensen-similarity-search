package bitbucket_test

import (
	"context"
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/models"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log/slog"
	"strings"
	"testing"
)

// ####################### tests
func TestDeleteObsoleteMarkdownsFromDatabase_CleansUpCorrectly(t *testing.T) {
	mockRepo := &mockRepository{
		foundContentIds: []uint{101, 102},
	}

	env := environment.Null()
	env.Repository = mockRepo

	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	dbMetas := []models.MarkdownMeta{
		{Model: models.Model{ID: 1}, Name: "foo"},
		{Model: models.Model{ID: 2}, Name: "bar"},
	}
	bitbucketMetas := []models.MarkdownMeta{
		{Model: models.Model{ID: 1}, Name: "foo"},
	}

	ctx := context.Background()
	err := hk.DeleteObsoleteMarkdownsFromDatabase(ctx, bitbucketMetas, dbMetas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []uint{2}
	got := mockRepo.deletedMetas
	if !cmp.Equal(got, want) {
		t.Errorf("deletedMetas mismatch:\n got:  %v\n want: %v", got, want)
	}

	want = []uint{101, 102}
	got = mockRepo.deletedContent
	if !cmp.Equal(got, want) {
		t.Errorf("deletedContent mismatch:\n got:  %v\n want: %v", got, want)
	}
}

func TestDeleteObsoleteMarkdowns_EarlyReturn(t *testing.T) {
	mockRepo := &mockRepository{}
	var core zapcore.Core

	env := &environment.Env{
		Repository: mockRepo,
		Logger: &logging.DefaultLogger{
			Logger: zap.New(core).Sugar(),
		},
	}
	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	err := hk.DeleteObsoleteMarkdownsFromDatabase(context.Background(),
		[]models.MarkdownMeta{{Name: "foo"}},
		[]models.MarkdownMeta{{Model: models.Model{ID: 1}, Name: "foo"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockRepo.deletedMetas) != 0 {
		t.Errorf("got deletedMetas = %v, want = []", mockRepo.deletedMetas)
	}
	if len(mockRepo.deletedContent) != 0 {
		t.Errorf("got deletedContent = %v, want = []", mockRepo.deletedContent)
	}
}

func TestDeleteObsoleteMarkdowns_ErrorFetchingContentIds(t *testing.T) {
	mockRepo := &mockRepository{
		findErr: errors.New("boom"),
	}
	var core zapcore.Core

	env := &environment.Env{
		Repository: mockRepo,
		Logger: &logging.DefaultLogger{
			Logger: zap.New(core).Sugar(),
		},
	}
	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	err := hk.DeleteObsoleteMarkdownsFromDatabase(context.Background(),
		[]models.MarkdownMeta{{Name: "foo"}},
		[]models.MarkdownMeta{{Model: models.Model{ID: 2}, Name: "bar"}},
	)

	if err == nil || !strings.Contains(err.Error(), "error fetching") {
		t.Errorf("expected fetch error, got = %v", err)
	}
}

func TestDeleteObsoleteMarkdowns_ErrorDeletingContent(t *testing.T) {
	mockRepo := &mockRepository{
		foundContentIds: []uint{101},
		deleteContErr:   errors.New("delete fail"),
	}
	var core zapcore.Core

	env := &environment.Env{
		Repository: mockRepo,
		Logger: &logging.DefaultLogger{
			Logger: zap.New(core).Sugar(),
		},
	}

	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	err := hk.DeleteObsoleteMarkdownsFromDatabase(context.Background(),
		[]models.MarkdownMeta{{Name: "foo"}},
		[]models.MarkdownMeta{{Model: models.Model{ID: 3}, Name: "zebra"}},
	)

	if err == nil || !strings.Contains(err.Error(), "delete fail") {
		t.Errorf("expected content deletion error, got = %v", err)
	}
}

func TestDeleteObsoleteMarkdowns_ErrorDeletingMetas(t *testing.T) {
	mockRepo := &mockRepository{
		foundContentIds: []uint{},
		deleteMetaErr:   errors.New("can't delete metas"),
	}
	var core zapcore.Core

	env := &environment.Env{
		Repository: mockRepo,
		Logger: &logging.DefaultLogger{
			Logger: zap.New(core).Sugar(),
		},
	}

	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	err := hk.DeleteObsoleteMarkdownsFromDatabase(context.Background(),
		[]models.MarkdownMeta{{Name: "foo"}},
		[]models.MarkdownMeta{{Model: models.Model{ID: 99}, Name: "baz"}},
	)

	if err == nil || !strings.Contains(err.Error(), "can't delete metas") {
		t.Errorf("expected meta deletion error, got = %v", err)
	}
}

func TestDeleteObsoleteMarkdowns_WarnWhenNoContentToDelete(t *testing.T) {
	mockRepo := &mockRepository{
		foundContentIds: []uint{},
	}
	var core zapcore.Core

	env := &environment.Env{
		Repository: mockRepo,
		Logger: &logging.DefaultLogger{
			Logger: zap.New(core).Sugar(),
		},
	}

	hk := &bitbucket.DefaultMarkdownHousekeeper{Env: env}

	err := hk.DeleteObsoleteMarkdownsFromDatabase(context.Background(),
		[]models.MarkdownMeta{{Name: "keep-me"}},
		[]models.MarkdownMeta{{Model: models.Model{ID: 7}, Name: "remove-me"}},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := mockRepo.deletedMetas; !cmp.Equal(got, []uint{7}) {
		t.Errorf("got deletedMetas = %v, want = [7]", got)
	}
	if got := mockRepo.deletedContent; len(got) != 0 {
		t.Errorf("expected no content deleted, got = %v", got)
	}
}

// ####################### creating mocks
type mockRepository struct {
	deletedMetas   []uint
	deletedContent []uint

	foundContentIds []uint
	findErr         error
	deleteMetaErr   error
	deleteContErr   error

	upsertMetasCalled         bool
	upsertContentsCalled      bool
	failMetaQuery             bool
	sanitizedName             string
	nameWasSanitizedCorrectly bool

	charCountByName map[string]uint
}

func (m *mockRepository) FindMarkdownsBySearchTermSimple(ctx context.Context, searchTerm string, markdowns *[]models.MarkdownContent) error {
	panic("implement me")
}

func (m *mockRepository) CountMarkdownsMatchesBySearchTermSimple(ctx context.Context, searchTerm string, matchCount *int) error {
	panic("implement me")
}

func (m *mockRepository) DeleteMarkdownMetasByIds(_ context.Context, ids []uint) error {
	if m.deleteMetaErr != nil {
		return m.deleteMetaErr
	}
	m.deletedMetas = ids
	return nil
}

func (m *mockRepository) DeleteMarkdownContentsByIds(_ context.Context, ids []uint) error {
	if m.deleteContErr != nil {
		return m.deleteContErr
	}
	m.deletedContent = ids
	return nil
}

func (m *mockRepository) FindUserLoginCredentials(_ context.Context, _ string, _ *models.User) error {
	return nil
}

func (m *mockRepository) FindAllMarkdownMetas(_ context.Context, _ *[]models.MarkdownMeta) error {
	if m.failMetaQuery {
		return m.findErr
	}
	return nil
}

func (m *mockRepository) FindMarkdownMetasWhereCharCountGreaterThan(_ context.Context, _ int, _ *[]models.MarkdownMeta) error {
	return nil
}

func (m *mockRepository) FindMarkdownContentByName(_ context.Context, _ string, _ *models.MarkdownContent) error {
	return nil
}

func (m *mockRepository) FindMarkdownContentIdsByMetaIds(_ context.Context, _ []uint, out *[]uint) error {
	if m.findErr != nil {
		return m.findErr
	}
	*out = m.foundContentIds
	return nil
}

func (m *mockRepository) UpsertMarkdownMetas(_ context.Context, metas []models.MarkdownMeta) error {
	if len(m.sanitizedName) > 0 {
		m.nameWasSanitizedCorrectly = metas[0].Name == m.sanitizedName
	}

	// if the map is filled in a test,
	// update the charCount values so you can verify the logic calling this receiver method
	if len(m.charCountByName) > 0 {
		for _, v := range metas {
			if _, ok := m.charCountByName[v.Name]; !ok {
				slog.Error(fmt.Sprintf("could not find matching meta key; key=%s", v.Name))
				continue
			}

			slog.Info(fmt.Sprintf("found matching meta key for %s", v.Name))
			slog.Info(fmt.Sprintf("updating char count to %d", v.CharCount))
			m.charCountByName[v.Name] = v.CharCount
		}
	}

	m.upsertMetasCalled = true
	return nil
}

func (m *mockRepository) UpsertMarkdownContents(_ context.Context, _ []models.MarkdownContent) error {
	m.upsertContentsCalled = true
	return nil
}
