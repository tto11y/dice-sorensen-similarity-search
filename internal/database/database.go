package database

import (
	"dice-sorensen-similarity-search/internal/config"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/models"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/url"
)

func InitDatabase(c *config.Configuration, l logging.Logger) (*gorm.DB, error) {
	l.LogInfo(nil, "Initializing Database")

	dsn := url.URL{
		User:     url.UserPassword(c.Database.Username, c.Database.Password),
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%d", c.Database.Host, c.Database.Port),
		Path:     c.Database.DatabaseName,
		RawQuery: (&url.Values{"sslmode": []string{"disable"}}).Encode(),
	}

	// PostgresSQL
	db, err := gorm.Open(
		postgres.Open(dsn.String()),
		&gorm.Config{Logger: logging.InitGormLogger(c)})

	if err != nil {
		l.LogErrorf(nil, "error initializing database: %v", err)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		l.LogErrorf(nil, "error setting connection properties on db conn pool")
		return nil, err
	}
	sqlDB.SetMaxIdleConns(c.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(c.Database.ConnMaxLifetime.Duration)

	l.LogDebug(nil, "connected to Database")

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		l.LogErrorf(nil, "error auto migrating models.User: %v", err)
		return nil, err
	}

	err = db.AutoMigrate(&models.MarkdownMeta{})
	if err != nil {
		l.LogErrorf(nil, "error auto migrating models.MarkdownContent: %v", err)
		return nil, err
	}

	err = db.AutoMigrate(&models.MarkdownContent{})
	if err != nil {
		l.LogErrorf(nil, "error auto migrating models.MarkdownContent: %v", err)
		return nil, err
	}

	return db, nil
}
