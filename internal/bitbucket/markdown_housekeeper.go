package bitbucket

import (
	"context"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/models"
	"dice-sorensen-similarity-search/internal/utils"
	"fmt"
	"strings"
	"time"
)

// MarkdownHousekeeper defines methods for cleaning up obsolete markdown records
type MarkdownHousekeeper interface {
	// DeleteObsoleteMarkdownsFromDatabase removes MarkdownMeta and MarkdownContent records
	// that no longer exist in the Bitbucket repository.
	//
	// param ctx param context.Context true "the context used for request-scoped operations"
	// param markdownMetasFromBitbucket the set of markdown meta records fetched from Bitbucket
	// param markdownMetasFromDb the set of markdown meta records currently in the database
	// return error if deletion or lookup operations fail
	DeleteObsoleteMarkdownsFromDatabase(ctx context.Context, markdownMetasFromBitbucket []models.MarkdownMeta, markdownMetasFromDb []models.MarkdownMeta) error
}

// DefaultMarkdownHousekeeper provides a default implementation of MarkdownHousekeeper.
type DefaultMarkdownHousekeeper struct {
	*environment.Env
}

// DeleteObsoleteMarkdownsFromDatabase compares markdown metadata from Bitbucket
// with entries in the database and deletes entries from the database that are no longer present.
//
// param ctx the context for database operations
// param markdownMetasFromBitbucket the current markdown metadata from Bitbucket
// param markdownMetasFromDb the existing markdown metadata in the database
// return error if any lookup or deletion fails
func (hk *DefaultMarkdownHousekeeper) DeleteObsoleteMarkdownsFromDatabase(ctx context.Context, markdownMetasFromBitbucket []models.MarkdownMeta, markdownMetasFromDb []models.MarkdownMeta) error {
	hk.LogInfo(nil, "start markdown meta data clean up")

	markdownMetasFromBitbucketByName := utils.SliceToMap(markdownMetasFromBitbucket, func(meta models.MarkdownMeta) string { return meta.Name })

	toBeDeletedMarkdownMetaIds := make([]uint, 0, len(markdownMetasFromDb)/2)
	for _, v := range markdownMetasFromDb {
		if _, ok := markdownMetasFromBitbucketByName[v.Name]; !ok {
			toBeDeletedMarkdownMetaIds = append(toBeDeletedMarkdownMetaIds, v.ID)
		}
	}

	if len(toBeDeletedMarkdownMetaIds) == 0 {
		hk.LogInfo(nil, "no cleanup for markdown files needed; early return")
		return nil
	}

	toBeDeletedMarkdownContentIds := make([]uint, 0, len(toBeDeletedMarkdownMetaIds))

	err := hk.FindMarkdownContentIdsByMetaIds(ctx, toBeDeletedMarkdownMetaIds, &toBeDeletedMarkdownContentIds)
	if err != nil {
		hk.LogError(nil, err.Error())
		return fmt.Errorf("error fetching to be deleted markdown content data from the database: %s", err.Error())
	}

	if len(toBeDeletedMarkdownContentIds) > 0 {
		err = hk.deleteObsoleteTuples(ctx, toBeDeletedMarkdownContentIds, MarkdownContent)
		if err != nil {
			return err
		}
	} else {
		msg := hk.createWarningMsgForMetasWithoutAReferenceToAContent(markdownMetasFromDb, toBeDeletedMarkdownMetaIds)
		hk.LogWarn(nil, msg)
	}

	err = hk.deleteObsoleteTuples(ctx, toBeDeletedMarkdownMetaIds, MarkdownMeta)
	if err != nil {
		return err
	}

	return nil
}

// deleteObsoleteTuples removes markdown records (either meta or content) from the database,
// logs the outcome and tracks the deletion duration.
//
// param ctx the operation context
// param toBeDeletedMarkdownTupleIds list of IDs to be deleted
// param modelType the type of model to delete (MarkdownMeta or MarkdownContent)
// return error if deletion fails or if modelType is invalid
func (hk *DefaultMarkdownHousekeeper) deleteObsoleteTuples(ctx context.Context, toBeDeletedMarkdownTupleIds []uint, modelType ModelType) error {
	start := time.Now()
	msg := fmt.Sprintf("deleting %d obsolete %s tuple(s)", len(toBeDeletedMarkdownTupleIds), modelType)

	hk.LogInfo(nil, "start "+msg)

	var err error
	switch modelType {
	case MarkdownContent:
		err = hk.DeleteMarkdownContentsByIds(ctx, toBeDeletedMarkdownTupleIds)
	case MarkdownMeta:
		err = hk.DeleteMarkdownMetasByIds(ctx, toBeDeletedMarkdownTupleIds)
	default:
		return fmt.Errorf("invalid model type: %s", modelType)
	}

	end := time.Now()

	if err != nil {
		hk.LogError(nil, err.Error())
		return fmt.Errorf("error deleting %s tuple(s) from the database: %s", modelType, err.Error())
	}

	hk.LogInfo(nil, "finished "+msg)
	hk.LogInfo(nil, fmt.Sprintf("duration: %dms", end.Sub(start).Milliseconds()))

	return nil
}

// createWarningMsgForMetasWithoutAReferenceToAContent generates a warning message for
// markdown meta entries that no longer have related content records.
//
// param markdownMetasFromDb all markdown metas in the database
// param toBeDeletedMarkdownMetaIds list of IDs with no associated markdown content
// return a formatted warning message listing the orphaned markdown meta entries
func (hk *DefaultMarkdownHousekeeper) createWarningMsgForMetasWithoutAReferenceToAContent(markdownMetasFromDb []models.MarkdownMeta, toBeDeletedMarkdownMetaIds []uint) string {
	markdownMetasFromDbById := utils.SliceToMap(markdownMetasFromDb, func(meta models.MarkdownMeta) uint { return meta.ID })

	var sb strings.Builder
	sb.WriteString("no markdown content is referenced by: ")

	for _, v := range toBeDeletedMarkdownMetaIds {
		meta := markdownMetasFromDbById[v]
		sb.WriteString(fmt.Sprintf("(id=%d, name=%s), ", v, meta.Name))
	}
	msg := sb.String()
	msg = strings.TrimSuffix(msg, ", ")

	return msg
}
