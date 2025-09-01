package bitbucket

import (
	"context"
	"dice-sorensen-similarity-search/internal/config"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"fmt"
	"github.com/gfleury/go-bitbucket-v1"
	"strings"
	"time"
)

// Result represents a wrapper for paginated file paths retrieved from the Bitbucket API.
//
// This struct contains a slice of generic values representing file path entries
// and any error encountered during retrieval.
type Result struct {
	AnyFilePaths []any
	Error        error
}

// BitbucketApiServiceAdapter wraps an abstraction layer around the Bitbucket APIs
// that are used within the application for retrieving repository content and file streams.
//
// @Summary Interface for Bitbucket API data access
type BitbucketApiServiceAdapter interface {
	GetContent(projectKey string, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error)
	GetRawContent(projectKey, repositorySlug, path string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error)
	StreamFiles(projectKey, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error)
}

// BitbucketApiClient is a concrete implementation of BitbucketApiServiceAdapter.
//
// It uses an embedded *bitbucketv1.APIClient to proxy requests and invert dependencies from the
// domain logic to the underlying API implementation.
//
// Based on the Dependency Inversion Principle [DIP].
//
// [DIP]: https://en.wikipedia.org/wiki/Dependency_inversion_principle
type BitbucketApiClient struct {
	*bitbucketv1.APIClient
}

func (a *BitbucketApiClient) GetContent(projectKey string, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return a.DefaultApi.GetContent(projectKey, repositorySlug, localVarOptionals)
}

func (a *BitbucketApiClient) GetRawContent(projectKey, repositorySlug, path string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return a.DefaultApi.GetRawContent(projectKey, repositorySlug, path, localVarOptionals)
}

func (a *BitbucketApiClient) StreamFiles(projectKey, repositorySlug string, localVarOptionals map[string]any) (*bitbucketv1.APIResponse, error) {
	return a.DefaultApi.StreamFiles(projectKey, repositorySlug, localVarOptionals)
}

// BitbucketReader defines methods for extracting content and structure
// from a Bitbucket repository relevant to Markdown documentation.
//
// @Summary Interface for reading Markdown content and structure from Bitbucket
type BitbucketReader interface {

	// ReadMarkdownFileStructureRecursively reads the full file structure of a repository,
	// returning only the Markdown (.md) files under the "markdowns/" root-level folder.
	//
	// Param projectName path string true "Bitbucket project key"
	// Param repoName path string true "Bitbucket repository name"
	// Param start query int false "Pagination start offset"
	// Param limit query int false "Pagination limit"
	ReadMarkdownFileStructureRecursively(projectName, repoName string, start, limit int) ([]string, error)

	// ReadRepoRootFolderContent returns names of all direct children (files and folders)
	// in the root folder of the specified Bitbucket repository.
	//
	// Param projectName path string true "Bitbucket project key"
	// Param repoName path string true "Bitbucket repository name"
	ReadRepoRootFolderContent(projectName, repoName string) ([]string, error)

	// ReadFileContentAtRevision reads the raw content of a single file at the specified revision.
	//
	// Param projectName path string true "Bitbucket project key"
	// Param repoName path string true "Bitbucket repository name"
	// Param filePath path string true "Path to file in repository"
	// Param revision query string true "Git reference or tag (e.g. 'main' or 'v1.0.0')"
	ReadFileContentAtRevision(projectName, repoName string, filePath string, revision string) (string, error)
}

// O11yBitbucketReader provides a concrete implementation of BitbucketReader that
// integrates Bitbucket API access with the Env abstraction.
//
// It supports recursive structure reading and markdown content extraction via an adapter.
type O11yBitbucketReader struct {
	*environment.Env
	Adapter BitbucketApiServiceAdapter
}

// ReadRepoRootFolderContent fetches the content of the given remote Bitbucket repository's root folder
//
// Returns a slice of top-level file/folder names or an error if extraction fails.
func (obbr *O11yBitbucketReader) ReadRepoRootFolderContent(projectName, repoName string) ([]string, error) {
	if obbr.Adapter == nil {
		return nil, fmt.Errorf("bitbucket API not initialized")
	}

	bitbucketResponse, err := obbr.Adapter.GetContent(projectName, repoName, nil)
	if err != nil {
		return nil, fmt.Errorf("error reading file structure from Bitbucket: %w", err)
	}

	if bitbucketResponse == nil {
		return nil, fmt.Errorf("bitbucket API response is nil")
	}

	if bitbucketResponse.Values == nil {
		return nil, fmt.Errorf("bitbucket API response has no values")
	}

	children, ok := bitbucketResponse.Values["children"]
	if !ok {
		return nil, fmt.Errorf("bitbucket API response does not contain children (i.e. files or folders)")
	}

	if children == nil {
		return []string{}, nil
	}

	anyChildren, ok := children.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("type conversion to map; received type: %T", children)
	}

	anyValues, ok := anyChildren["values"].([]any)
	if !ok {
		return nil, fmt.Errorf("type conversion to slice of type any failed; received type: %T", children)
	}

	var rootFolderChildren []string
	for _, v := range anyValues {
		vm, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("type conversion to map failed; received type: %T", v)
		}

		p, ok := vm["path"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("type conversion to map failed; received type: %T", v)
		}

		n, ok := p["name"].(string)
		if !ok {
			return nil, fmt.Errorf("type conversion to string failed; received type: %T", v)
		}

		rootFolderChildren = append(rootFolderChildren, n)
	}

	return rootFolderChildren, nil
}

// ReadFileContentAtRevision retrieves the raw contents of the specified file at a given revision.
//
// Note: the revision string may be a tag or commit hash.
func (obbr *O11yBitbucketReader) ReadFileContentAtRevision(projectName, repoName, filePath, revision string) (string, error) {
	if obbr.Adapter == nil {
		return "", fmt.Errorf("bitbucket API not initialized")
	}

	params := make(map[string]any)
	// todo use the revision
	//params["at"] = fmt.Sprintf("refs/tags/%v", revision)

	bitbucketResponse, err := obbr.Adapter.GetRawContent(projectName, repoName, filePath, params)
	if err != nil {
		return "", err
	}

	if bitbucketResponse == nil {
		return "", fmt.Errorf("bitbucket API response is nil")
	}

	return string(bitbucketResponse.Payload), nil
}

// ReadMarkdownFileStructureRecursively recursively traverses the Bitbucket repository,
// collecting absolute paths of all .md files located under the root-level `markdowns/` directory.
//
// Only Markdown (.md) files are included. Returns a list of file paths or an error.
func (obbr *O11yBitbucketReader) ReadMarkdownFileStructureRecursively(projectName, repoName string, start, limit int) ([]string, error) {
	if obbr.Adapter == nil {
		return nil, fmt.Errorf("bitbucket API not initialized")
	}

	m := make(map[string]any)
	m["start"] = start
	m["limit"] = limit

	read := func() <-chan Result {
		outStream := make(chan Result)

		go func() {
			defer close(outStream)

			var reachedLastPage bool

			for !reachedLastPage {
				s := time.Now()
				obbr.LogInfo(nil, "start fetching file structure")
				bitbucketResponse, err := obbr.Adapter.StreamFiles(projectName, repoName, m)
				e := time.Now()
				obbr.LogInfo(nil, fmt.Sprintf("fetched file structure in %v", e.Sub(s)))

				if err != nil {
					outStream <- Result{AnyFilePaths: nil, Error: fmt.Errorf("error reading file structure from Bitbucket: %w", err)}
					return
				}

				if bitbucketResponse == nil || bitbucketResponse.Values == nil {
					outStream <- Result{AnyFilePaths: nil, Error: fmt.Errorf("bitbucket API response is nil or has no paged values")}
					return
				}

				values, ok := bitbucketResponse.Values["values"]
				if !ok {
					outStream <- Result{AnyFilePaths: nil, Error: fmt.Errorf("bitbucket API response does not contain the property 'values'")}
					reachedLastPage = true
					return
				}

				if values == nil {
					outStream <- Result{AnyFilePaths: []any{}, Error: nil}
					reachedLastPage = true
					continue
				}

				anyFilePaths, ok := values.([]any)
				if !ok {
					outStream <- Result{AnyFilePaths: nil, Error: fmt.Errorf("type conversion to slice of type any failed; received type: %T", values)}
					reachedLastPage = true
					continue
				}

				outStream <- Result{AnyFilePaths: anyFilePaths, Error: nil}

				isLastPage, lastPageOk := bitbucketResponse.Values["isLastPage"].(bool)
				if !lastPageOk {
					obbr.LogWarn(nil, "bitbucket API response does not contain property 'isLastPage'")
					break
				}

				if isLastPage {
					obbr.LogInfo(nil, "reached last page")
					break
				}

				nextPageStart, nextPageOk := bitbucketResponse.Values["nextPageStart"].(float64)
				if !nextPageOk {
					obbr.LogWarn(nil, "bitbucket API response does not contain property 'nextPageStart'")
					break
				}

				m["start"] = int(nextPageStart)
			}
		}()

		return outStream
	}

	consume := func(results <-chan Result) ([]string, error) {
		var filePaths []string

		for result := range results {

			if result.Error != nil {
				return nil, result.Error
			}

			for _, v := range result.AnyFilePaths {
				fp, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("type conversion to slice of type string failed; received type: %T", v)
				}

				if !strings.HasPrefix(fp, "markdowns/") {
					continue
				}

				filePaths = append(filePaths, fp)
			}
		}

		return filePaths, nil
	}

	results := read()
	return consume(results)
}

// InitBitbucket initializes the Bitbucket API with the provided configuration and creates an instance of *O11yBitbucketReader
func InitBitbucket(c *config.Configuration, env *environment.Env) (*O11yBitbucketReader, error) {
	env.LogInfo(logging.GetLogTypeInitialization(), "initializing Bitbucket API (async)")

	if c.BitBucket.Url == nil {
		return nil, fmt.Errorf("bitbucket url is not set")
	}

	bitbucketConfig := bitbucketv1.Configuration{
		BasePath:  c.BitBucket.Url.String(),
		Host:      c.BitBucket.Url.Host,
		Scheme:    c.BitBucket.Url.Scheme,
		UserAgent: "cim-api",
	}

	ctx := context.Background()

	if len(c.BitBucket.AccessToken) > 0 {
		ctx = context.WithValue(ctx, bitbucketv1.ContextAccessToken, c.BitBucket.AccessToken)

	} else if len(c.BitBucket.User) > 0 && len(c.BitBucket.Password) > 0 {
		ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, bitbucketv1.BasicAuth{
			UserName: c.BitBucket.User,
			Password: c.BitBucket.Password,
		})
	}

	bitbucketApi := bitbucketv1.NewAPIClient(ctx, &bitbucketConfig)

	_, err := bitbucketApi.DefaultApi.GetPullRequestCount()
	if err != nil {
		return nil, err
	}

	env.LogDebug(logging.GetLogTypeInitialization(), "Bitbucket API initialized")

	return &O11yBitbucketReader{env, &BitbucketApiClient{bitbucketApi}}, nil
}
