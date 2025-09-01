package markdowndoc

import (
	"dice-sorensen-similarity-search/internal/api"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/models"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"slices"
)

// Api defines HTTP endpoints for accessing markdown content and navigation metadata.
type Api interface {
	GetNavigationItemsTrees(c *gin.Context)
	GetMarkdownByName(c *gin.Context)
	GetMarkdownSearchTermMatches(c *gin.Context)
}

// Controller handles API operations related to markdown metadata and content.
//
// @Summary Markdown navigation and content controller
type Controller struct {
	*environment.Env
	NavigationItemTreeService
	MarkdownSearchMatchMapper
}

type MatchesWithSimilarity struct {
	content    models.MarkdownContent
	similarity float64
}

// GetNavigationItemsTrees returns the navigation structure for all available markdown files.
//
// @ID getNavigationItemTrees
// @Summary Get navigation item trees for markdown files
// @Tags navigation
// @Router /markdown-doc/navigation-items [get]
// @Success	200	{object} api.RestJsonResponse{data=[]markdowndoc.NavigationItem}
// @Failure 500
func (hc *Controller) GetNavigationItemsTrees(c *gin.Context) {
	ctx := c.Request.Context()

	var markdownMetas []models.MarkdownMeta
	if err := hc.FindMarkdownMetasWhereCharCountGreaterThan(ctx, 0, &markdownMetas); err != nil {
		hc.LogError(logging.GetLogType("markdown-doc"), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("error reading markdown meta info: %s", err.Error()))
		return
	}

	trees := hc.BuildNavigationItemTrees(markdownMetas)
	c.JSON(http.StatusOK, trees)
}

// GetMarkdownByName returns the markdown content associated with the provided name.
//
// @ID getMarkdownByName
// @Summary Get markdown content by file name
// @Tags markdown
// @Router /markdown-doc/markdown/{name} [get]
// @Param name path string true "Markdown file name without extension"
// @Success 200 {object} map[string]string "Returns markdown content"
// @Failure 400
// @Failure 500
func (hc *Controller) GetMarkdownByName(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("name")
	if len(name) <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse("path variable 'name' is missing"))
		return
	}

	var markdownContent models.MarkdownContent
	err := hc.FindMarkdownContentByName(ctx, name, &markdownContent)
	if err != nil {
		hc.LogError(logging.GetLogType("markdown-doc"), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponsef("error reading markdown meta info: %s", err))
		return
	}

	response := struct {
		Content string `json:"content"`
	}{
		Content: markdownContent.Content,
	}
	c.JSON(http.StatusOK, response)
}

func (hc *Controller) GetMarkdownSearchTermMatches(c *gin.Context) {
	ctx := c.Request.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		msg := fmt.Sprintf("error while reading request body: %s", err)
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(msg))
		return
	}

	var payload MarkdownSearchPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		msg := fmt.Sprintf("error while unmarshaling request body: %s", err)
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(msg))
		return
	}

	if len(payload.Term) <= 0 {
		msg := "did not perform search because no search term was present"
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(msg))
		return
	}

	pageSize := 5
	if payload.Pageable.PageSize > 0 {
		pageSize = payload.Pageable.PageSize
	}

	searchMatches := make([]models.MarkdownContent, 0)
	err = hc.FindMarkdownsBySearchTermSimple(ctx, payload.Term, &searchMatches)
	if err != nil {
		msg := fmt.Sprintf("error reading Markdown search matches: %s", err)
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponse(msg))
		return
	}

	matchesWithSimilarity := make([]MatchesWithSimilarity, 0, pageSize)
	for i, v := range searchMatches {
		if i >= pageSize {
			break
		}

		s := TrigramSorensenDiceSimilarity(v.Content, payload.Term)
		matchesWithSimilarity = append(matchesWithSimilarity, MatchesWithSimilarity{content: v, similarity: s})
	}

	// sorts matches based on similarity in descending order (the most similar match is the first element)
	slices.SortFunc(matchesWithSimilarity, func(a, b MatchesWithSimilarity) int {
		if a.similarity > b.similarity {
			return -1
		}
		return 1
	})

	firstPage := make([]models.MarkdownContent, 0, pageSize)
	for _, v := range matchesWithSimilarity {
		firstPage = append(firstPage, v.content)
	}

	var matchCount int
	err = hc.CountMarkdownsMatchesBySearchTermSimple(ctx, payload.Term, &matchCount)
	if err != nil {
		msg := fmt.Sprintf("error counting Markdown search matches: %s", err)
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponse(msg))
		return
	}

	page, err := hc.mapToMarkdownSearchPage(payload, pageSize, matchCount, firstPage)
	if err != nil {
		msg := fmt.Sprintf("error mapping to page response: %s", err)
		hc.LogError(logging.GetLogType("markdown-doc"), msg)
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.NewErrorResponse(msg))
		return
	}

	c.JSON(http.StatusOK, page)
}
