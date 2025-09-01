package database_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"dice-sorensen-similarity-search/internal/database"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/models"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
	"moul.io/zapgorm2"
	"os"
	"testing"
	"time"
)

var env *environment.Env
var sqlMock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	mockedGormDb, sqlDb, s, err := initMockedDatabase()
	if err != nil {
		return
	}

	defer func(mockDb *sql.DB) {
		sqlMock.ExpectClose()
		cErr := mockDb.Close()

		if cErr != nil {
			slog.Error(fmt.Sprintf("close database error: %v", cErr))
			return
		}
	}(sqlDb)

	// set up the environment
	sqlMock = s
	env = environment.Null()

	env.Repository = &database.GormRepository{DB: mockedGormDb}

	code := m.Run()

	os.Exit(code)
}

func initMockedDatabase() (*gorm.DB, *sql.DB, sqlmock.Sqlmock, error) {
	mockDb, sqlM, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{Logger: setupGormLogger()})

	if err != nil {
		slog.Error(fmt.Sprintf("error initializing database: %v", err))
		return nil, nil, nil, fmt.Errorf("error initializing database: %v", err)
	}

	return db, mockDb, sqlM, nil
}

func setupGormLogger() zapgorm2.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	gormW := zapcore.AddSync(&lumberjack.Logger{
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	gormCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		gormW,
		zapcore.DebugLevel,
	)
	zapGormLogger := zap.New(gormCore)
	gormLogger := zapgorm2.New(zapGormLogger)
	gormLogger.SetAsDefault()

	return gormLogger
}

func parseTime(value string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05.999999 -07:00", value)
	if err != nil {
		panic(err)
	}
	return t
}

// ####################### GormRepository
func TestGormRepository_FindAllMarkdownMetas(t *testing.T) {
	markdownMetaRows := sqlMock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"name",
		"path",
	})

	want := []models.MarkdownMeta{
		{Model: models.Model{ID: 3, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "1-Onboarding", Path: "markdowns/Gateway"},
		{Model: models.Model{ID: 4, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "2-Data-Preparation", Path: "markdowns/Gateway"},
		{Model: models.Model{ID: 5, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "3-Visualization", Path: "markdowns/Gateway"},
		{Model: models.Model{ID: 12, CreatedAt: parseTime("2025-05-28 07:12:16.133174 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "4-Technical-Docs", Path: "markdowns/Gateway"},
		{Model: models.Model{ID: 59, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "5-OpenTelemetry", Path: "markdowns/Gateway"},
		{Model: models.Model{ID: 6, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "GitHub-Flavored-Markdown", Path: "markdowns/GitHub-Flavored-Markdown"},
		{Model: models.Model{ID: 61, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "Getting-Started", Path: "markdowns/Guidelines"},
	}

	for _, r := range want {
		markdownMetaRows.AddRow(
			r.ID,
			r.CreatedAt,
			r.UpdatedAt,
			r.Name,
			r.Path,
		)
	}

	// NOTE: ExpectedQuery expects a regex string as param
	sqlMock.ExpectQuery("^SELECT \\* FROM \"markdown_meta\"").
		WillReturnRows(markdownMetaRows)

	var got []models.MarkdownMeta
	err := env.FindAllMarkdownMetas(context.Background(), &got)
	if err != nil {
		t.Fatalf("FindAllMarkdownMetas error: %v", err)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}

	// make them unequal
	want[3] = got[0]

	if cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}
}

func TestGormRepository_FindMarkdownMetasWhereCharCountGreaterThan(t *testing.T) {
	markdownMetaRows := sqlMock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"name",
		"path",
		"char_count",
	})

	testMetas := []models.MarkdownMeta{
		{Model: models.Model{ID: 4, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "2-Data-Preparation", Path: "markdowns/Gateway", CharCount: 5},
		{Model: models.Model{ID: 5, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "3-Visualization", Path: "markdowns/Gateway", CharCount: 5},
		{Model: models.Model{ID: 59, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "5-OpenTelemetry", Path: "markdowns/Gateway", CharCount: 5},
		{Model: models.Model{ID: 6, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "GitHub-Flavored-Markdown", Path: "markdowns/GitHub-Flavored-Markdown", CharCount: 5},
		{Model: models.Model{ID: 61, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "Getting-Started", Path: "markdowns/Guidelines", CharCount: 5},
		{Model: models.Model{ID: 3, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "1-Onboarding", Path: "markdowns/Gateway", CharCount: 0},
		{Model: models.Model{ID: 12, CreatedAt: parseTime("2025-05-28 07:12:16.133174 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "4-Technical-Docs", Path: "markdowns/Gateway", CharCount: 0},
	}

	want := testMetas[:len(testMetas)-2]

	for _, r := range want {
		markdownMetaRows.AddRow(
			r.ID,
			r.CreatedAt,
			r.UpdatedAt,
			r.Name,
			r.Path,
			r.CharCount,
		)
	}

	// NOTE: ExpectedQuery expects a regex string as param
	sqlMock.ExpectQuery("^SELECT \\* FROM \"markdown_meta\" WHERE char_count > \\$1").
		WillReturnRows(markdownMetaRows)

	var got []models.MarkdownMeta
	err := env.FindMarkdownMetasWhereCharCountGreaterThan(context.Background(), 2, &got)
	if err != nil {
		t.Fatalf("FindAllMarkdownMetas error: %v", err)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}

	if cmp.Equal(testMetas, got) {
		t.Error(cmp.Diff(testMetas, got))
		return
	}
}

func TestGormRepository_DeleteMarkdownMetasByIds(t *testing.T) {
	sqlMock.ExpectExec("^DELETE FROM markdown_meta WHERE id IN \\(\\$1,\\$2,\\$3\\)").
		WithArgs(3, 4, 5).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := env.DeleteMarkdownMetasByIds(context.Background(), []uint{3, 4, 5})
	if err != nil {
		t.Fatalf("DeleteMarkdownMetasByIds error: %v", err)
	}
}

func TestGormRepository_DeleteMarkdownContentsByIds(t *testing.T) {
	sqlMock.ExpectExec("^DELETE FROM markdown_contents WHERE id IN \\(\\$1,\\$2,\\$3\\)").
		WithArgs(1, 2, 3).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := env.DeleteMarkdownContentsByIds(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("DeleteMarkdownContentsByIds error: %v", err)
	}
}

func TestGormRepository_FindUserLoginCredentials(t *testing.T) {

	want := models.User{
		Model:    models.Model{ID: 1},
		Username: "username",
		Password: "hashed_password",
		Email:    "test@email.com",
	}

	sqlMock.ExpectQuery("^SELECT \\* FROM \"users\" WHERE username = \\$1 LIMIT \\$2").
		WillReturnRows(sqlMock.
			NewRows([]string{"id", "username", "email", "password"}).
			AddRow(1, want.Username, want.Email, want.Password),
		)

	got := models.User{}

	err := env.FindUserLoginCredentials(context.Background(), "testuser", &got)
	if err != nil {
		t.Fatalf("FindUserLoginCredentials error: %v", err)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}
}

func TestGormRepository_FindMarkdownContentByName(t *testing.T) {

	want := models.MarkdownContent{
		Model:  models.Model{ID: 1},
		MetaID: 61,
		Meta:   models.MarkdownMeta{
			//Name: "Getting-Started",
			//Path: "markdowns/Getting-Started",
		},
		Content: "# Heading 1\n\n### Heading 3",
	}

	sqlMock.ExpectQuery("^SELECT .* FROM \"markdown_contents\" LEFT JOIN \"markdown_meta\" \"Meta\" ON \"markdown_contents\"\\.\"meta_id\" = \"Meta\"\\.\"id\" WHERE name = \\$1 ORDER BY \"markdown_contents\"\\.\"id\" LIMIT \\$2").
		//WithArgs("Getting-Started").
		WillReturnRows(sqlMock.
			NewRows([]string{"id", "meta_id", "content"}).
			AddRow(want.ID, want.MetaID, want.Content))

	got := models.MarkdownContent{}
	err := env.FindMarkdownContentByName(context.Background(), "Getting-Started", &got)
	if err != nil {
		t.Fatalf("FindMarkdownContentByName error: %v", err)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}
}

func TestGormRepository_FindMarkdownContentIdsByMetaIds(t *testing.T) {
	wantIds := []uint{10, 11, 12}

	sqlMock.ExpectQuery("^SELECT id FROM markdown_contents WHERE meta_id IN \\(\\$1,\\$2,\\$3\\)").
		WithArgs(3, 4, 5).
		WillReturnRows(sqlMock.
			NewRows([]string{"id"}).
			AddRow(wantIds[0]).
			AddRow(wantIds[1]).
			AddRow(wantIds[2]),
		)

	var gotIds []uint
	err := env.FindMarkdownContentIdsByMetaIds(context.Background(), []uint{3, 4, 5}, &gotIds)
	if err != nil {
		t.Fatalf("FindMarkdownContentIdsByMetaIds error: %v", err)
	}

	if !cmp.Equal(wantIds, gotIds) {
		t.Error(cmp.Diff(wantIds, gotIds))
		return
	}
}

func TestGormRepository_UpsertMarkdownMetas(t *testing.T) {
	want := []models.MarkdownMeta{
		{Model: models.Model{ID: 3, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "1-Onboarding", Path: "markdowns/Gateway", CharCount: 1234},
		{Model: models.Model{ID: 4, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "2-Data-Preparation", Path: "markdowns/Gateway", CharCount: 1234},
		{Model: models.Model{ID: 5, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "3-Visualization", Path: "markdowns/Gateway", CharCount: 1234},
		{Model: models.Model{ID: 12, CreatedAt: parseTime("2025-05-28 07:12:16.133174 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "4-Technical-Docs", Path: "markdowns/Gateway", CharCount: 1234},
		{Model: models.Model{ID: 59, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "5-OpenTelemetry", Path: "markdowns/Gateway", CharCount: 1234},
		{Model: models.Model{ID: 6, CreatedAt: parseTime("2025-05-27 10:06:56.823450 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "GitHub-Flavored-Markdown", Path: "markdowns/GitHub-Flavored-Markdown", CharCount: 1234},
		{Model: models.Model{ID: 61, CreatedAt: parseTime("2025-06-05 06:40:25.891387 +00:00"), UpdatedAt: parseTime("2025-06-18 09:22:38.894670 +00:00")}, Name: "Getting-Started", Path: "markdowns/Guidelines", CharCount: 1234},
	}

	args := flattenMarkdownMetas(want)
	// GORM appends an updated_at property; it's exact value cannot be anticipated
	// since it's a timestamp created on execution.
	// therefore, we have to accept/expect any argument at the end of the arguments slice
	args = append(args, sqlmock.AnyArg())

	rows := sqlmock.NewRows([]string{"id"})
	for _, m := range want {
		rows.AddRow(m.Model.ID)
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("^INSERT INTO \"markdown_meta\" \\(\"created_at\",\"updated_at\",\"name\",\"path\",\"char_count\",\"id\"\\) VALUES .* ON CONFLICT \\(\"name\"\\) DO UPDATE SET .* RETURNING \"id\"").
		WithArgs(args...).
		WillReturnRows(rows)
	sqlMock.ExpectCommit()

	err := env.UpsertMarkdownMetas(context.Background(), want)
	if err != nil {
		t.Fatalf("UpsertMarkdownMetas error: %v", err)
	}
}

func flattenMarkdownMetas(metas []models.MarkdownMeta) []driver.Value {
	args := make([]driver.Value, 0, len(metas))
	for _, m := range metas {
		args = append(args, m.CreatedAt, m.UpdatedAt, m.Name, m.Path, m.CharCount, m.ID)
	}

	return args
}

func TestGormRepository_UpsertMarkdownContents(t *testing.T) {
	want := []models.MarkdownContent{
		{
			Model:   models.Model{ID: 1, CreatedAt: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 0, 0, 0, time.UTC)},
			Content: "# Introduction\nThis is the intro content.",
			MetaID:  3,
		},
		{
			Model:   models.Model{ID: 2, CreatedAt: time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 5, 0, 0, time.UTC)},
			Content: "## Setup\nEnsure all dependencies are installed.",
			MetaID:  4,
		},
		{
			Model:   models.Model{ID: 3, CreatedAt: time.Date(2025, 6, 3, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 10, 0, 0, time.UTC)},
			Content: "**Data Prep:** Clean and normalize your data.",
			MetaID:  5,
		},
		{
			Model:   models.Model{ID: 4, CreatedAt: time.Date(2025, 6, 4, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 15, 0, 0, time.UTC)},
			Content: "`Technical note:` Use consistent data formats.",
			MetaID:  12,
		},
		{
			Model:   models.Model{ID: 5, CreatedAt: time.Date(2025, 6, 5, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 20, 0, 0, time.UTC)},
			Content: "Enable OpenTelemetry tracing in config.",
			MetaID:  59,
		},
		{
			Model:   models.Model{ID: 6, CreatedAt: time.Date(2025, 6, 6, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 25, 0, 0, time.UTC)},
			Content: "Use `~~~` for GitHub-flavored fenced code blocks.",
			MetaID:  6,
		},
		{
			Model:   models.Model{ID: 7, CreatedAt: time.Date(2025, 6, 7, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2025, 6, 18, 9, 30, 0, 0, time.UTC)},
			Content: "Welcome to the Getting Started guide!",
			MetaID:  61,
		},
	}

	args := flattenMarkdownContents(want)
	// GORM appends an updated_at property; it's exact value cannot be anticipated
	// since it's a timestamp created on execution.
	// therefore, we have to accept/expect any argument at the end of the arguments slice
	args = append(args, sqlmock.AnyArg())

	rows := sqlmock.NewRows([]string{"id"})
	for _, c := range want {
		rows.AddRow(c.Model.ID)
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("^INSERT INTO \"markdown_contents\" \\(\"created_at\",\"updated_at\",\"content\",\"meta_id\",\"id\"\\) VALUES .* ON CONFLICT \\(\"meta_id\"\\) DO UPDATE SET .*").
		WithArgs(args...).
		WillReturnRows(rows)
	sqlMock.ExpectCommit()

	err := env.UpsertMarkdownContents(context.Background(), want)
	if err != nil {
		t.Fatalf("UpsertMarkdownContents error: %v", err)
	}
}

func flattenMarkdownContents(contents []models.MarkdownContent) []driver.Value {
	args := make([]driver.Value, 0, len(contents))
	for _, c := range contents {
		args = append(args, c.CreatedAt, c.UpdatedAt, c.Content, c.MetaID, c.ID)
	}

	return args
}

// ####################### NullRepository
func TestNullRepository_DeleteMarkdownMetasByIds(t *testing.T) {
	repo := &database.NullRepository{}
	err := repo.DeleteMarkdownMetasByIds(context.Background(), []uint{1, 2, 3})
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_DeleteMarkdownContentsByIds(t *testing.T) {
	repo := &database.NullRepository{}
	err := repo.DeleteMarkdownContentsByIds(context.Background(), []uint{4, 5, 6})
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_FindUserLoginCredentials(t *testing.T) {
	repo := &database.NullRepository{}
	var user models.User
	err := repo.FindUserLoginCredentials(context.Background(), "testuser", &user)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_FindAllMarkdownMetas(t *testing.T) {
	repo := &database.NullRepository{}
	var markdownMetas []models.MarkdownMeta
	err := repo.FindAllMarkdownMetas(context.Background(), &markdownMetas)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_FindMarkdownContentByName(t *testing.T) {
	repo := &database.NullRepository{}
	var markdownContent models.MarkdownContent
	err := repo.FindMarkdownContentByName(context.Background(), "test-markdown", &markdownContent)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_FindMarkdownContentIdsByMetaIds(t *testing.T) {
	repo := &database.NullRepository{}
	var markdownContentIds []uint
	err := repo.FindMarkdownContentIdsByMetaIds(context.Background(), []uint{7, 8, 9}, &markdownContentIds)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_UpsertMarkdownMetas(t *testing.T) {
	repo := &database.NullRepository{}
	err := repo.UpsertMarkdownMetas(context.Background(), []models.MarkdownMeta{})
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}

func TestNullRepository_UpsertMarkdownContents(t *testing.T) {
	repo := &database.NullRepository{}
	err := repo.UpsertMarkdownContents(context.Background(), []models.MarkdownContent{})
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
		return
	}
}
