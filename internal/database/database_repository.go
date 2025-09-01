package database

import (
	"context"
	"dice-sorensen-similarity-search/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// Repository defines data access methods for interacting with Markdown-related
// database records, including Markdown metadata, content, and user login credentials.
//
// @Summary Interface for Markdown data storage operations
type Repository interface {

	// DeleteMarkdownMetasByIds deletes Markdown meta records with the given IDs.
	//
	// Param metaIds body []uint true "List of Markdown meta IDs to delete"
	DeleteMarkdownMetasByIds(ctx context.Context, metaIds []uint) error

	// DeleteMarkdownContentsByIds deletes markdown content records with the given IDs.
	//
	// Param contentIds body []uint true "List of Markdown content IDs to delete"
	DeleteMarkdownContentsByIds(ctx context.Context, contentIds []uint) error

	// FindUserLoginCredentials fetches the user record with the specified username.
	//
	// Param username path string true "Username"
	FindUserLoginCredentials(ctx context.Context, username string, user *models.User) error

	// FindAllMarkdownMetas retrieves all Markdown metadata records from the database.
	FindAllMarkdownMetas(ctx context.Context, markdownMetas *[]models.MarkdownMeta) error

	// FindMarkdownMetasWhereCharCountGreaterThan retrieves all Markdown metadata records from the database.
	FindMarkdownMetasWhereCharCountGreaterThan(ctx context.Context, x int, markdownMetas *[]models.MarkdownMeta) error

	// FindMarkdownContentByName fetches Markdown content by file name.
	//
	// Param name path string true "Markdown file name"
	FindMarkdownContentByName(ctx context.Context, name string, markdownContents *models.MarkdownContent) error

	// FindMarkdownContentIdsByMetaIds fetches content IDs by related Markdown meta IDs.
	//
	// Param metaIds body []uint true "Meta IDs to search"
	FindMarkdownContentIdsByMetaIds(ctx context.Context, markdownMetaIds []uint, markdownContentIds *[]uint) error

	FindMarkdownsBySearchTermSimple(ctx context.Context, searchTerm string, markdowns *[]models.MarkdownContent) error

	CountMarkdownsMatchesBySearchTermSimple(ctx context.Context, searchTerm string, matchCount *int) error

	// UpsertMarkdownMetas inserts or updates Markdown meta records.
	//
	// Param markdownMetas body []models.MarkdownMeta true "Markdown meta data"
	UpsertMarkdownMetas(ctx context.Context, markdownMetas []models.MarkdownMeta) error

	// UpsertMarkdownContents inserts or updates Markdown content records.
	//
	// Param markdownContents body []models.MarkdownContent true "Markdown content data"
	UpsertMarkdownContents(ctx context.Context, markdownContents []models.MarkdownContent) error
}

// NullRepository is a no-op implementation of the Repository interface.
// Useful for testing or default wiring when no database operations are required.
type NullRepository struct{}

func (n *NullRepository) DeleteMarkdownMetasByIds(ctx context.Context, metaIds []uint) error {
	return nil
}

func (n *NullRepository) DeleteMarkdownContentsByIds(ctx context.Context, contentIds []uint) error {
	return nil
}

func (n *NullRepository) FindUserLoginCredentials(ctx context.Context, username string, user *models.User) error {
	return nil
}

func (n *NullRepository) FindAllMarkdownMetas(ctx context.Context, markdownMetas *[]models.MarkdownMeta) error {
	return nil
}

func (n *NullRepository) FindMarkdownMetasWhereCharCountGreaterThan(ctx context.Context, x int, markdownMetas *[]models.MarkdownMeta) error {
	return nil
}

func (n *NullRepository) FindMarkdownContentByName(ctx context.Context, name string, markdownContents *models.MarkdownContent) error {
	return nil
}

func (n *NullRepository) FindMarkdownContentIdsByMetaIds(ctx context.Context, markdownMetaIds []uint, markdownContentIds *[]uint) error {
	return nil
}

func (n *NullRepository) FindMarkdownsBySearchTermSimple(ctx context.Context, searchTerm string, markdowns *[]models.MarkdownContent) error {
	return nil
}

func (n *NullRepository) CountMarkdownsMatchesBySearchTermSimple(ctx context.Context, searchTerm string, matchCount *int) error {
	return nil
}

func (n *NullRepository) UpsertMarkdownMetas(ctx context.Context, markdownMetas []models.MarkdownMeta) error {
	return nil
}

func (n *NullRepository) UpsertMarkdownContents(ctx context.Context, markdownContents []models.MarkdownContent) error {
	return nil
}

// ensure GormRepository implements Repository
var _ Repository = &NullRepository{}

// GormRepository provides a GORM-based implementation of the Repository interface.
type GormRepository struct {
	*gorm.DB
}

// ensure GormRepository implements Repository
var _ Repository = &GormRepository{}

func (g *GormRepository) DeleteMarkdownMetasByIds(ctx context.Context, metaIds []uint) error {
	return g.DB.
		WithContext(ctx).
		Exec("DELETE FROM markdown_meta WHERE id IN ?", metaIds).
		Error
}

func (g *GormRepository) DeleteMarkdownContentsByIds(ctx context.Context, contentIds []uint) error {
	return g.DB.
		WithContext(ctx).
		Exec("DELETE FROM markdown_contents WHERE id IN ?", contentIds).
		Error
}

func (g *GormRepository) FindAllMarkdownMetas(ctx context.Context, markdownMetas *[]models.MarkdownMeta) error {
	return g.DB.
		WithContext(ctx).
		Find(markdownMetas).
		Error
}

func (g *GormRepository) FindMarkdownMetasWhereCharCountGreaterThan(ctx context.Context, x int, markdownMetas *[]models.MarkdownMeta) error {
	return g.DB.
		WithContext(ctx).
		Where("char_count > ?", x).
		Find(markdownMetas).
		Error
}

func (g *GormRepository) FindUserLoginCredentials(ctx context.Context, username string, user *models.User) error {
	return g.DB.
		WithContext(ctx).
		Model(models.User{}).
		Where("username = ?", username).
		Take(user).
		Error
}

func (g *GormRepository) FindMarkdownContentByName(ctx context.Context, name string, markdownContent *models.MarkdownContent) error {
	return g.DB.
		WithContext(ctx).
		Model(&markdownContent).
		Joins("Meta").
		First(&markdownContent, "name = ?", name).
		Error
}

func (g *GormRepository) FindMarkdownContentIdsByMetaIds(ctx context.Context, markdownMetaIds []uint, markdownContentIds *[]uint) error {
	return g.DB.
		WithContext(ctx).
		Preload(clause.Associations).
		Raw("SELECT id FROM markdown_contents WHERE meta_id IN ?", markdownMetaIds).
		Scan(&markdownContentIds).
		Error
}

func (g *GormRepository) FindMarkdownsBySearchTermSimple(ctx context.Context, searchTerm string, markdowns *[]models.MarkdownContent) error {

	var markdownJoined []struct {
		MetaID        uint
		MetaCreatedAt time.Time
		MetaUpdatedAt time.Time
		Name          string
		Path          string
		CharCount     uint

		ContentId        uint
		ContentCreatedAt time.Time
		ContentUpdatedAt time.Time
		Content          string
	}

	err := g.DB.
		WithContext(ctx).
		Raw(`
				SELECT
					mm.id AS MetaID, 
				    mm.created_at AS MetaCreatedAt, 
				    mm.updated_at AS MetaUpdatedAt, 
				    mm.name AS Name, 
				    mm.path AS Path, 
				    mm.char_count AS CharCount,
				    mc.id AS ContentId, 
				    mc.created_at AS ContentCreatedAt, 
				    mc.updated_at AS ContentUpdatedAt, 
				    mc.content AS Content
				FROM markdown_contents mc
				JOIN markdown_meta mm ON mm.id = mc.meta_id
				WHERE content LIKE '%'|| ? ||'%' 
					AND path NOT LIKE 'markdowns/.%'`,
			searchTerm,
		).
		Scan(&markdownJoined).
		Error
	if err != nil {
		return err
	}

	if len(markdownJoined) == 0 {
		return nil
	}

	for _, m := range markdownJoined {
		meta := models.MarkdownMeta{
			Path: m.Path,
			Name: m.Name,
		}

		content := models.MarkdownContent{
			Meta:    meta,
			Content: m.Content,
		}

		*markdowns = append(*markdowns, content)
	}

	return nil
}

func (g *GormRepository) CountMarkdownsMatchesBySearchTermSimple(ctx context.Context, searchTerm string, matchCount *int) error {
	return g.DB.
		WithContext(ctx).
		Raw(`
				SELECT count(*)
				FROM markdown_contents mc,
					 markdown_meta mm
				WHERE mc.meta_id = mm.id
					AND content LIKE '%'|| ? ||'%' 
					AND path NOT LIKE 'markdowns/.%'`,
			searchTerm,
		).
		Scan(matchCount).
		Error
}

func (g *GormRepository) UpsertMarkdownMetas(ctx context.Context, markdownMetas []models.MarkdownMeta) error {
	return g.DB.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			// update all columns to new value on `name` conflict except primary keys
			// and those columns having default values from sql func
			Columns:   []clause.Column{{Name: "name"}},
			UpdateAll: true,
		}).
		Create(&markdownMetas).
		Error
}

func (g *GormRepository) UpsertMarkdownContents(ctx context.Context, markdownContents []models.MarkdownContent) error {
	return g.DB.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			// update all columns to new value on `meta_id` conflict except primary keys
			// and those columns having default values from sql func
			Columns:   []clause.Column{{Name: "meta_id"}},
			UpdateAll: true,
		}).
		Create(&markdownContents).
		Error
}
