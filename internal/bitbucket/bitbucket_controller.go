package bitbucket

import (
	"context"
	"dice-sorensen-similarity-search/internal/api"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
	"strings"
)

// Api defines the set of endpoints related to synchronizing markdown files from Bitbucket.
//
// @Summary API contract for Bitbucket-based markdown ingestion
type Api interface {
	// FetchMarkdownsFromBitbucket imports markdown files from a Bitbucket repository into the database.
	FetchMarkdownsFromBitbucket(c *gin.Context)
}

// Controller handles the ingestion of markdown documents from Bitbucket repositories.
// It coordinates between the BitbucketReader, Repository, and cleanup logic to maintain content freshness.
type Controller struct {
	*environment.Env
	BitbucketReader
	MarkdownHousekeeper

	RepositoryName string
	ProjectName    string
}

// ensure Controller implements Api
var _ Api = &Controller{}

type ModelType string

const (
	MarkdownMeta    ModelType = "markdown meta"
	MarkdownContent ModelType = "markdown content"
)

// FetchMarkdownsFromBitbucket retrieves Markdown file paths and contents from a Bitbucket repository,
// deduplicates and stores them into the database, and deletes obsolete entries.
// Only `.md` files under the "markdowns/" folder are processed. Filenames containing spaces or dots
// are sanitized before insertion.
//
// @ID fetchMarkdownsFromBitbucket
// @Summary Sync Markdown files from Bitbucket into the database
// @Tags bitbucket
// @Router /bitbucket/markdowns/ [get]
// @Success 204
// @Failure 400
// @Failure 500
func (bc *Controller) FetchMarkdownsFromBitbucket(c *gin.Context) {
	var ctx context.Context
	if c.Request == nil || c.Request.Context() == nil {
		ctx = context.Background()
	} else {
		ctx = c.Request.Context()
	}

	filePaths, err := bc.ReadMarkdownFileStructureRecursively(bc.ProjectName, bc.RepositoryName, 0, 150)
	if err != nil {
		bc.LogError(nil, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("Error reading filePath: %s", err.Error()))
		return
	}

	var markdownMetasFromBitbucket []models.MarkdownMeta
	var markdownContentsFromBitbucket []models.MarkdownContent

	for _, filePath := range filePaths {
		fileContent, err := bc.ReadFileContentAtRevision(bc.ProjectName, bc.RepositoryName, filePath, "0")
		if err != nil {
			bc.LogError(nil, err.Error())
		}

		extension := filepath.Ext(filePath)
		if extension != ".md" {
			bc.LogWarn(nil, fmt.Sprintf("file extension is not markdown: %s", filePath))
			continue
		}
		name := strings.TrimSuffix(filepath.Base(filePath), extension)
		path := filepath.Dir(filePath)

		if strings.Contains(name, " ") {
			name = strings.ReplaceAll(name, " ", "_")
		}

		var charCount uint
		if len(fileContent) > 0 {
			charCount = uint(len(fileContent))
		}

		markdownMetasFromBitbucket = append(markdownMetasFromBitbucket, models.MarkdownMeta{Name: name, Path: path, CharCount: charCount})
		markdownContentsFromBitbucket = append(markdownContentsFromBitbucket, models.MarkdownContent{Content: fileContent})
	}

	var markdownMetasFromDb []models.MarkdownMeta

	err = bc.FindAllMarkdownMetas(ctx, &markdownMetasFromDb)
	if err != nil {
		bc.LogError(nil, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("error fetching existing markdown meta data from the database: %s", err.Error()))
		return
	}

	if len(markdownMetasFromDb) > 0 {
		err := bc.DeleteObsoleteMarkdownsFromDatabase(ctx, markdownMetasFromBitbucket, markdownMetasFromDb)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponse(err.Error()))
			return
		}
	}

	err = bc.UpsertMarkdownMetas(ctx, markdownMetasFromBitbucket)
	if err != nil {
		bc.LogError(nil, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("error writing markdown meta data into the database: %s", err.Error()))
		return
	}

	// links meta and content
	for i := 0; i < len(markdownMetasFromBitbucket); i++ {
		markdownContentsFromBitbucket[i].Meta = markdownMetasFromBitbucket[i]
	}

	err = bc.UpsertMarkdownContents(ctx, markdownContentsFromBitbucket)
	if err != nil {
		bc.LogError(nil, err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("error writing markdown files into the database: %s", err.Error()))
		return
	}

	c.JSON(http.StatusNoContent, "")
}
