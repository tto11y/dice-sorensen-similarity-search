package markdowndoc

import (
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/models"
	"dice-sorensen-similarity-search/internal/utils"
	"fmt"
	"github.com/samborkent/uuidv7"
	"golang.org/x/text/collate"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type MarkdownSearchPayload struct {
	Term     string
	Pageable Pageable
}

type MarkdownSearchMatch struct {
	Href            string `json:"href"`
	Path            string `json:"path"`
	PrettyPath      string `json:"prettyPath"`
	Label           string `json:"label"`
	MatchingText    string `json:"matchingText"`
	TextBeforeMatch string `json:"textBeforeMatch"`
	TextAfterMatch  string `json:"textAfterMatch"`
}

type Page[T any] struct {
	TotalElements int      `json:"totalElements"`
	TotalPages    int      `json:"totalPages"`
	Content       []T      `json:"content"`
	Pageable      Pageable `json:"pageable"`
}

type Pageable struct {
	PageNumber int  `json:"pageNumber"`
	PageSize   int  `json:"pageSize"`
	Sort       Sort `json:"sort"`
}

type Sort struct {
	defaultDirection Direction `json:"-"`
	Orders           []Order   `json:"orders"`
}

func (s *Sort) DefaultDirection() Direction {
	return s.defaultDirection
}

func NewSort(orders []Order) Sort {
	return Sort{defaultDirection: ASC, Orders: orders}
}

type Order struct {
	Property  string    `json:"property"`
	Direction Direction `json:"direction"`
}

type Direction string

const (
	ASC  Direction = "ASC"
	DESC Direction = "DESC"
)

type NavigationItem struct {
	Uuid     string            `json:"uuid"`
	Href     string            `json:"href"`
	Label    string            `json:"label"`
	Parent   *NavigationItem   `json:"-"`
	Children []*NavigationItem `json:"children"`
}

type NavigationItemTreeService struct {
	*environment.Env
	*collate.Collator
}

type MarkdownSearchMatchMapper struct {
	*environment.Env
}

// itemTreesLister implements the interface [collate.Lister]
// which can be passed into the receiver method Sort() of [collate.Collator]
//
// A [collate.Collator] allows to define the [collation order], so locale-aware sorting of characters is possible.
//
// Instead of Go's default pure Unicode code point ordering, [collate.Collator] is used to provide lexicographic order
// with locale-aware sorting (like filesystems do) for the Markdown doc's navigation items.
//
// For example, Go's default pure Unicode code point ordering would sort the strings below as follows:
//
//		"A-Guidelines"
//	 "ABC"
//		"A_Guidelines"
//	 "Gateway"
//	 "Guidelines"
//	 "Z_Guidelines"
//
// While the locale-aware [collate.Collator] sorts it like this (which is more natural):
//
//		"A_Guidelines"
//		"A-Guidelines"
//	 "ABC"
//	 "Gateway"
//	 "Guidelines"
//	 "Z_Guidelines"
//
// [collation order]: https://en.wikipedia.org/wiki/Collation
type itemTreesLister struct {
	itemTrees []*NavigationItem
}

func (l itemTreesLister) Len() int {
	return len(l.itemTrees)
}

func (l itemTreesLister) Swap(i, j int) {
	temp := l.itemTrees[i]
	l.itemTrees[i] = l.itemTrees[j]
	l.itemTrees[j] = temp
}

func (l itemTreesLister) Bytes(i int) []byte {
	// returns the bytes of Href at index i
	return []byte(l.itemTrees[i].Href)
}

// BuildNavigationItemTrees constructs a hierarchical navigation structure from a slice of MarkdownMeta objects.
//
// It parses metadata paths, removes the "markdowns/" prefix, and assembles nested navigation trees,
// linking items by parent-child relationships.
// The function ensures correct parent-child relationships and links items under the same parent.
//
// @ID buildNavigationTrees
// @Summary Build markdown navigation trees from metadata
// @Param markdownMetas body []models.MarkdownMeta true "List of markdown metadata items"
// @Return A slice containing the root navigation items with their complete tree structure.
func (n NavigationItemTreeService) BuildNavigationItemTrees(markdownMetas []models.MarkdownMeta) []*NavigationItem {

	var rootNavigationItems []*NavigationItem

	for _, v := range markdownMetas {
		if len(v.Path) <= 0 {
			n.LogErrorf(nil, fmt.Sprintf("markdown meta %s has empty path", v.Name))
			continue
		}

		pathElements := strings.Split(v.Path, "/")
		if !strings.HasPrefix(pathElements[0], "markdowns") {
			continue
		}

		// Top-level Markdowns (i.e., Markdown files residing under the markdowns/ folder in Bitbucket)
		// must be added directly to the root navigation items since their Path is "markdowns".
		// In other words, splitting and truncating their Path won't work
		// as for Markdown files residing under some folder that is beneath the folder markdowns/
		if len(pathElements) == 1 {
			// The landing page needs special handling in the frontend.
			// Therefore, it should not be part of the side navigation items.
			if v.Name == "Landing-Page" {
				n.LogDebugf(nil, "skip processing top-level element: %s", v.Name)
				continue
			}

			n.LogDebugf(nil, "processing top-level element w/o children: %s", v.Name)

			root := NavigationItem{
				Uuid:  uuidv7.New().String(),
				Label: strings.ReplaceAll(v.Name, "_", " "),
				Href:  v.Name,
			}
			rootNavigationItems = append(rootNavigationItems, &root)

			continue
		}

		// removes "markdowns" from path elements (this is only supposed for non-top-level files)
		pathElements = pathElements[1:]

		parent := NavigationItem{
			Uuid:  uuidv7.New().String(),
			Label: strings.ReplaceAll(pathElements[0], "_", " "),
			Href:  pathElements[0],
		}

		bottomToRootTree := n.createNavItemTree(pathElements[1:], &parent)

		bottomMostNavItem := NavigationItem{
			Uuid:  uuidv7.New().String(),
			Label: strings.ReplaceAll(v.Name, "_", " "),
			Href:  v.Name,
		}

		bottomToRootTree.Children = append(bottomToRootTree.Children, &bottomMostNavItem)

		root := n.findRoot(bottomToRootTree)

		rootNavigationItems = append(rootNavigationItems, root)
	}

	rootNavigationItems = n.linkChildrenWithTheSameParent(rootNavigationItems)

	visibleRootNavigationItems := n.removeDotPrefixedRoots(rootNavigationItems)
	if visibleRootNavigationItems != nil {
		rootNavigationItems = visibleRootNavigationItems
	}

	// sort roots based on top-level Href
	l := itemTreesLister{itemTrees: rootNavigationItems}
	n.Sort(l)

	n.removeNumberPrefixFromRoots(rootNavigationItems)

	return rootNavigationItems
}

// createNavItemTree recursively assembles a navigation tree from a sequence of path elements.
//
// It constructs parent-child relationships by creating a new NavigationItem for each path element.
// While descending from the provided parent node, the recursion continues until all path elements are processed.
//
// ID createNavTreeBranch
// Summary Build a branch of the navigation tree
// Param pathElements body []string true "Path components"
// Param navItem body NavigationItem true "Parent navigation item"
// Return The bottom-most child containing a recursive parent tree (from bottom to top)
func (n NavigationItemTreeService) createNavItemTree(pathElements []string, navItem *NavigationItem) *NavigationItem {

	// early return in case the path has only one element
	if len(pathElements) == 0 && navItem.Parent == nil {
		return navItem
	}

	if len(pathElements) == 0 {
		return navItem
	}

	child := &NavigationItem{
		Uuid:   uuidv7.New().String(),
		Label:  strings.ReplaceAll(pathElements[0], "_", " "),
		Href:   pathElements[0],
		Parent: navItem,
	}

	navItem.Children = append(navItem.Children, child)

	return n.createNavItemTree(pathElements[1:], child)
}

// findRoot traverses upward in a navigation tree to find the root element
//
// This function follows parent pointers recursively until it finds the top-most parent with no parent.
//
// ID findNavigationRoot
// Summary Traverse a navigation item tree to find its root
// Param navItemPtr body NavigationItem true "Starting navigation node"
// Return The root element of the navigation tree
func (n NavigationItemTreeService) findRoot(navItemPtr *NavigationItem) *NavigationItem {
	if navItemPtr.Parent == nil {
		return navItemPtr
	}

	return n.findRoot(navItemPtr.Parent)
}

// linkChildrenWithTheSameParent merges navigation trees that share the same parent Href
// eliminating duplicates.
//
// It ensures that navigation items with identical Href values are combined under a single parent,
// eliminating redundant entries while preserving hierarchical relationships.
//
// ID linkSiblingNavigationItems
// Summary Merge navigation items with the same parent into unified subtrees
// Param navigationItemTrees body []NavigationItem true "Navigation trees to merge"
// Return A consolidated list of root navigation items with children correctly linked
func (n NavigationItemTreeService) linkChildrenWithTheSameParent(navigationItemTrees []*NavigationItem) []*NavigationItem {

	if len(navigationItemTrees) == 0 {
		return navigationItemTrees
	}

	treesByHrefAndUuid := utils.SliceToMap(navigationItemTrees, func(navigationItem *NavigationItem) string {
		return navigationItem.Href + ":" + navigationItem.Uuid
	})

	uuidsByHref := make(map[string][]string)
	for _, v := range navigationItemTrees {
		if uuids, ok := uuidsByHref[v.Href]; ok {
			uuidsByHref[v.Href] = append(uuids, v.Uuid)
			continue
		}

		uuidsByHref[v.Href] = []string{v.Uuid}
	}

	for _, self := range navigationItemTrees {
		uuids, ok := uuidsByHref[self.Href]
		// jump to the next iteration
		// if self.Href references an already processed (and removed) navigation item
		if !ok {
			continue
		}

		for _, uuid := range uuids {
			// jump to the next iteration to prevent linking self's children twice
			if self.Uuid == uuid {
				continue
			}

			key := self.Href + ":" + uuid
			twin, ok := treesByHrefAndUuid[key]
			if !ok {
				n.LogErrorf(logging.GetLogType("markdown-doc"), "could not find a navigation item tree for %s; please check if the map was created correctly", key)
				continue
			}

			// links children into one slice and removes twin
			self.Children = append(self.Children, twin.Children...)
			delete(treesByHrefAndUuid, key)
		}

		// removes the map entry of the href that was currently processed
		delete(uuidsByHref, self.Href)
	}

	navigationItemTrees = make([]*NavigationItem, 0, len(treesByHrefAndUuid))
	for _, v := range treesByHrefAndUuid {
		navigationItemTrees = append(navigationItemTrees, v)
	}

	sort.Slice(navigationItemTrees, func(a, b int) bool {
		return strings.Compare(navigationItemTrees[a].Href, navigationItemTrees[b].Href) == -1
	})

	// traverses the children of each tree recursively (down to the bottom-most children of each tree)
	for i := 0; i < len(navigationItemTrees); i++ {
		navigationItemTrees[i].Children = n.linkChildrenWithTheSameParent(navigationItemTrees[i].Children)
	}

	return navigationItemTrees
}

// removeDotPrefixedRoots removes roots (including their children) whose Href are prefixed with a dot (.)
// It runs in O(n) time.
//
// ID removeDotPrefixedRoots
// Param rootNavigationItems body []*NavigationItem true "root navigation items"
func (n NavigationItemTreeService) removeDotPrefixedRoots(rootNavigationItems []*NavigationItem) []*NavigationItem {
	compiledRegex, err := regexp.Compile("^\\.")
	if err != nil {
		n.LogErrorf(nil, "failed to compile regex: %s", err.Error())
		return nil
	}

	visibleRootNavigationItems := make([]*NavigationItem, 0, len(rootNavigationItems))
	for _, v := range rootNavigationItems {
		// skip a hidden top-level element
		if compiledRegex.MatchString(v.Href) {
			continue
		}
		visibleRootNavigationItems = append(visibleRootNavigationItems, v)
	}

	return visibleRootNavigationItems
}

// removeNumberPrefixFromRoots removes number prefixes from a root's Label and Href properties
//
// ID removeNumberPrefixFromRoots
// Param rootNavigationItems body []*NavigationItem true "root navigation items"
func (n NavigationItemTreeService) removeNumberPrefixFromRoots(rootNavigationItems []*NavigationItem) {
	compiledRegex, err := regexp.Compile("^\\d+")
	if err != nil {
		n.LogErrorf(nil, "failed to compile regex: %s", err.Error())
		return
	}

	hrefSeparator := "_"
	labelSeparator := " "
	for _, root := range rootNavigationItems {

		hasChildren := len(root.Children) > 0
		isPrefixedHref := compiledRegex.MatchString(root.Href)
		isPrefixedLabel := compiledRegex.MatchString(root.Label)

		if hasChildren && isPrefixedHref && isPrefixedLabel {
			prefix := compiledRegex.FindString(root.Href)
			root.Href = strings.TrimPrefix(root.Href, prefix+hrefSeparator)
			root.Label = strings.TrimPrefix(root.Label, prefix+labelSeparator)
			continue
		}

		// for top-level Markdown files, we must not remove the prefix from the Href.
		// otherwise, the frontend cannot navigate to/query the correct Markdown file
		if !hasChildren && isPrefixedHref && isPrefixedLabel {
			prefix := compiledRegex.FindString(root.Href)
			root.Label = strings.TrimPrefix(root.Label, prefix+labelSeparator)
			continue
		}

		if isPrefixedHref != isPrefixedLabel {
			n.LogErrorf(nil, "Href (%s) and Label (%s) differ, but should not (except the separator)", root.Href, root.Label)
			n.LogError(nil, "a bug in the logic was detected")
		}
	}
}

func (m MarkdownSearchMatchMapper) mapToMarkdownSearchPage(payload MarkdownSearchPayload, pageSize, matchCount int, searchMatches []models.MarkdownContent) (Page[MarkdownSearchMatch], error) {
	if searchMatches == nil {
		return Page[MarkdownSearchMatch]{}, fmt.Errorf("search matches must not be nil")
	}

	matches := make([]MarkdownSearchMatch, 0, len(searchMatches))
	for _, v := range searchMatches {
		pathElements := strings.Split(v.Meta.Path, "/")
		// removes "markdowns" from path elements (this is only supposed for non-top-level files)
		pathElements = pathElements[1:]
		path := strings.Join(pathElements, "/")

		label := v.Meta.Name
		if len(path) == 0 {
			compiledRegex, err := regexp.Compile("^\\d+")
			if err != nil {
				m.LogErrorf(nil, "failed to compile regex: %s", err.Error())
				return Page[MarkdownSearchMatch]{}, err
			}

			if compiledRegex.MatchString(label) {
				m.LogInfo(nil, "we don't have a path; but it's a top-level element => so we remove the number prefix for the label")
				hrefSeparator := "_"
				prefix := compiledRegex.FindString(label)

				trimmedName := strings.TrimPrefix(label, prefix+hrefSeparator)
				label = trimmedName

				m.LogDebugf(nil, "removed prefix (%s) from path root: %s", prefix, label)
			}
		}

		match := MarkdownSearchMatch{
			Label:        strings.ReplaceAll(label, "_", " "),
			Href:         v.Meta.Name,
			Path:         path,
			PrettyPath:   strings.ReplaceAll(path, "_", " "),
			MatchingText: payload.Term,
		}

		matches = append(matches, match)
	}

	m.removeNumberPrefixFromPathRoots(matches)
	m.removeNumberPrefixFromPrettyPathRoots(matches)

	totalPages := utils.CalculateTotalPages(matchCount, pageSize)

	orders := make([]Order, 0)
	orders = append(orders, Order{Property: "similarity", Direction: DESC})
	payload.Pageable.Sort.Orders = orders

	page := Page[MarkdownSearchMatch]{
		Content:       matches,
		Pageable:      payload.Pageable,
		TotalElements: matchCount,
		TotalPages:    totalPages,
	}

	return page, nil
}

// removeNumberPrefixFromPathRoots removes numeric prefixes from the root elements
// of paths in a slice of MarkdownSearchMatch.
//
// A numeric prefix is defined as a sequence of digits at the beginning of the root
// path element, followed by an underscore (e.g., "123_root"). If such a prefix is
// found, it is removed, and the path is updated accordingly.
//
// For example:
//
//	Input:  "123_root/section"
//	Output: "root/section"
//
// Parameters:
//   - markdownSearchMatches: A slice of MarkdownSearchMatch structs whose Path fields
//     may be modified in-place.
func (m MarkdownSearchMatchMapper) removeNumberPrefixFromPathRoots(markdownSearchMatches []MarkdownSearchMatch) {
	compiledRegex, err := regexp.Compile("^\\d+")
	if err != nil {
		m.LogErrorf(nil, "failed to compile regex: %s", err.Error())
		return
	}

	hrefSeparator := "_"
	for i := 0; i < len(markdownSearchMatches); i++ {
		pathElements := strings.Split(markdownSearchMatches[i].Path, "/")
		root := pathElements[0]

		if !compiledRegex.MatchString(root) {
			continue
		}

		prefix := compiledRegex.FindString(root)

		trimmedRoot := strings.TrimPrefix(root, prefix+hrefSeparator)
		pathElements[0] = trimmedRoot

		markdownSearchMatches[i].Path = strings.Join(pathElements, "/")

		m.LogDebugf(nil, "removed prefix (%s) from path root: %s", prefix, root)
	}
}

func (m MarkdownSearchMatchMapper) removeNumberPrefixFromPrettyPathRoots(markdownSearchMatches []MarkdownSearchMatch) {
	compiledRegex, err := regexp.Compile("^\\d+")
	if err != nil {
		m.LogErrorf(nil, "failed to compile regex: %s", err.Error())
		return
	}

	hrefSeparator := " "
	for i := 0; i < len(markdownSearchMatches); i++ {
		pathElements := strings.Split(markdownSearchMatches[i].PrettyPath, "/")
		prettyRoot := pathElements[0]

		if !compiledRegex.MatchString(prettyRoot) {
			continue
		}

		prefix := compiledRegex.FindString(prettyRoot)

		trimmedPrettyRoot := strings.TrimPrefix(prettyRoot, prefix+hrefSeparator)
		pathElements[0] = trimmedPrettyRoot

		markdownSearchMatches[i].PrettyPath = strings.Join(pathElements, "/")

		m.LogDebugf(nil, "removed prefix (%s) from pretty path root: %s", prefix, prettyRoot)
	}
}

func (m MarkdownSearchMatchMapper) removeMatchesWithDotPrefixedPath(markdownSearchMatches []MarkdownSearchMatch) []MarkdownSearchMatch {
	compiledRegex, err := regexp.Compile("^\\.")
	if err != nil {
		m.LogErrorf(nil, "failed to compile regex: %s", err.Error())
		return nil
	}

	visibleMarkdownSearchMatches := make([]MarkdownSearchMatch, 0, len(markdownSearchMatches))
	for _, v := range markdownSearchMatches {
		// skip a hidden top-level element
		if compiledRegex.MatchString(v.Path) {
			continue
		}
		visibleMarkdownSearchMatches = append(visibleMarkdownSearchMatches, v)
	}

	return visibleMarkdownSearchMatches
}

func TrigramSorensenDiceSimilarity(a, b string) float64 {

	aTrigrams := TransformToUniqueTrigrams(a)
	bTrigrams := TransformToUniqueTrigrams(b)

	aCount, bCount := len(aTrigrams), len(bTrigrams)
	aTrigramsByTrigram := make(map[string]struct{}, len(aTrigrams))
	for _, v := range aTrigrams {
		aTrigramsByTrigram[v] = struct{}{}
	}

	var intersectionCount int
	for _, bT := range bTrigrams {
		if _, ok := aTrigramsByTrigram[bT]; !ok {
			continue
		}
		intersectionCount++
	}

	// Sorensen-Dice coefficient
	//   SDC = 2 * |A ∩ B| / (|A| + |B|)
	return 2 * float64(intersectionCount) / float64(aCount+bCount)
}

func TransformToUniqueTrigrams(a string) []string {
	if len(a) == 0 {
		return []string{}
	}

	// split on non-word characters
	re := regexp.MustCompile(`\W+`)
	words := re.Split(a, -1)

	var trigramCount int
	for _, word := range words {
		// 1 there's always one trigram because of padding
		// 2 aside from the initial trigram, we need to shift left n times
		//   with n equal to the count of character in the string (=> len(a))
		trigramCount += 1 + len(word)
	}

	// to minimize the memory footprint, we use struct as value
	uniqueTrigrams := make(map[string]struct{}, trigramCount)

	for _, word := range words {
		word = strings.ToLower(word)
		padded := "  " + word + " "

		for i := 0; i < 1+len(word); i++ {
			t := padded[:3]
			uniqueTrigrams[t] = struct{}{}
			padded = padded[1:]
		}
	}

	trigrams := make([]string, 0, len(uniqueTrigrams))
	for t := range uniqueTrigrams {
		trigrams = append(trigrams, t)
	}

	// the following quicksort runs in n*lg(n) on average
	// because we can assume that the input is randomly ordered (=not sorted)
	slices.Sort(trigrams)

	return trigrams
}
