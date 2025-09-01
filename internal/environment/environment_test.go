package environment_test

import (
	"dice-sorensen-similarity-search/internal/database"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
	"testing"
)

func TestEnvironment_Null(t *testing.T) {
	env := environment.Null()

	if env == nil {
		t.Fatalf("Null environment returned nil")
	}

	switch env.Repository.(type) {
	case *database.NullRepository:
	default:
		t.Errorf("want repository to be NullRepository, got %v", env.Repository)
	}

	switch env.Logger.(type) {
	case *logging.NullLogger:
	default:
		t.Errorf("want logger to be NullLogger, got %v", env.Logger)
	}
}

func TestEnvironment(t *testing.T) {
	db, err := setupMockedDb(t)
	if err != nil {
		t.Error(err)
		return
	}

	var core zapcore.Core
	logger := zap.New(core)

	env := environment.Environment(
		&database.GormRepository{DB: db},
		&logging.DefaultLogger{Logger: logger.Sugar()},
	)

	switch env.Repository.(type) {
	case *database.GormRepository:
	default:
		t.Errorf("want repository to be GormRepository, got %v", env.Repository)
	}

	switch env.Logger.(type) {
	case *logging.DefaultLogger:
	default:
		t.Errorf("want logger to be DefaultLogger, got %v", env.Logger)
	}
}

func setupMockedDb(t *testing.T) (*gorm.DB, error) {
	mockDb, _, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{Logger: setupGormLogger()})
	if err != nil {
		t.Fatalf("Error initializing mocked database: %v", err)
		return nil, err
	}

	return db, nil
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
