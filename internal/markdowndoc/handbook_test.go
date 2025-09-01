package markdowndoc_test

import (
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/markdowndoc"
	"dice-sorensen-similarity-search/internal/models"
	"dice-sorensen-similarity-search/internal/utils"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNavigationItemHierarchy(t *testing.T) {
	tests := []struct {
		name          string
		markdownMetas []models.MarkdownMeta
		expectedTree  map[string]struct {
			children  []string
			hasPrefix bool
		}
	}{
		{
			name: "Proper parent-child linking",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway/SubFolder"},
			},
			expectedTree: map[string]struct {
				children  []string
				hasPrefix bool
			}{
				"Gateway":   {children: []string{"SubFolder"}, hasPrefix: false},
				"SubFolder": {children: []string{"File2"}, hasPrefix: false},
			},
		},
		{
			name: "No duplicate children",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway"},
			},
			expectedTree: map[string]struct {
				children  []string
				hasPrefix bool
			}{
				"Gateway": {children: []string{"File1", "File2"}, hasPrefix: false},
			},
		},
		{
			name: "Deep nested structure",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Root/Level1/Level2"},
				{Name: "File2", Path: "markdowns/Root/Level1"},
			},
			expectedTree: map[string]struct {
				children  []string
				hasPrefix bool
			}{
				"Root":   {children: []string{"Level1"}, hasPrefix: false},
				"Level1": {children: []string{"Level2"}, hasPrefix: false},
				"Level2": {children: []string{"File1"}, hasPrefix: false},
			},
		},
		{
			name: "Deep nested structure with top-level elements",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Root/Level1/Level2"},
				{Name: "File2", Path: "markdowns/Root/Level1"},
				{Name: "Top_Level_Element", Path: "markdowns"},
				{Name: "1_Prefixed_Top_Level_Element", Path: "markdowns"},
			},
			expectedTree: map[string]struct {
				children  []string
				hasPrefix bool
			}{
				"Root":                         {children: []string{"Level1"}, hasPrefix: false},
				"Level1":                       {children: []string{"Level2"}, hasPrefix: false},
				"Level2":                       {children: []string{"File1"}, hasPrefix: false},
				"Top_Level_Element":            {children: []string{}, hasPrefix: false},
				"1_Prefixed_Top_Level_Element": {children: []string{}, hasPrefix: true},
			},
		},
		{
			name: "Deep nested structure with landing page",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Root/Level1/Level2"},
				{Name: "File2", Path: "markdowns/Root/Level1"},
				{Name: "Landing_Page", Path: "markdowns"},
			},
			expectedTree: map[string]struct {
				children  []string
				hasPrefix bool
			}{
				"Root":   {children: []string{"Level1"}, hasPrefix: false},
				"Level1": {children: []string{"Level2"}, hasPrefix: false},
				"Level2": {children: []string{"File1"}, hasPrefix: false},
			},
		},
	}

	c := collate.New(language.English)
	env := environment.Null()
	env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}
	s := markdowndoc.NavigationItemTreeService{Env: env, Collator: c}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			navigationTrees := s.BuildNavigationItemTrees(tt.markdownMetas)
			navigationTreesByHref := utils.SliceToMap(navigationTrees, func(i *markdowndoc.NavigationItem) string { return i.Href })

			// validate parent-child relationships
			treeMap := make(map[string][]string)

			var traverse func(*markdowndoc.NavigationItem)
			traverse = func(node *markdowndoc.NavigationItem) {
				if node.Children != nil {
					for _, child := range node.Children {
						treeMap[node.Href] = append(treeMap[node.Href], child.Href)
						traverse(child)
					}
				}
			}

			for _, root := range navigationTrees {
				// add empty slice of children for top-level elements
				if root.Children == nil {
					treeMap[root.Href] = []string{}
					continue
				}

				traverse(root)
			}

			if _, ok := treeMap["Landing-Page"]; ok {
				t.Error("Landing-Page must not be part of the navigation trees slice")
				return
			}

			// verify that expected parents have correct children
			for expectedRootHref, expectedProperties := range tt.expectedTree {
				actualChildren, ok := treeMap[expectedRootHref]
				if !ok {
					t.Errorf("parent %s not found in tree", expectedRootHref)
					return
				}

				if len(expectedProperties.children) == 0 {
					fmt.Println("top-level element:", expectedRootHref)

					compiledRegex, err := regexp.Compile("^\\d+")
					if err != nil {
						t.Errorf("failed to compile regex: %s", err)
						return
					}

					tree, ok := navigationTreesByHref[expectedRootHref]
					if !ok {
						t.Errorf("parent %s not found in tree", expectedRootHref)
						return
					}

					if expectedProperties.hasPrefix {
						isPrefixedHref := compiledRegex.MatchString(tree.Href)
						if !isPrefixedHref {
							t.Errorf("for top-level Markdown want Href to be prefixed, got %s", tree.Href)
							return
						}

						isPrefixedLabel := compiledRegex.MatchString(tree.Label)
						if isPrefixedLabel {
							t.Errorf("for top-level Markdown want Label to NOT be prefixed, got %s", tree.Label)
							return
						}
					} else {
						if strings.ReplaceAll(tree.Href, "_", " ") != tree.Label {
							t.Errorf("for top-level Markdown want Href to equal Label, got Href=%s and Label=%s", tree.Href, tree.Label)
							return
						}
					}
				}

				// ensure no duplicates and children match expectations
				expectedChildrenSet := make(map[string]bool)
				for _, child := range expectedProperties.children {
					expectedChildrenSet[child] = true
				}

				actualChildrenSet := make(map[string]bool)
				for _, child := range actualChildren {
					actualChildrenSet[child] = true
				}

				for child := range expectedChildrenSet {
					if !actualChildrenSet[child] {
						t.Errorf("expected child %s missing under parent %s", child, expectedRootHref)
						return
					}
				}
			}
		})
	}
}

func TestBuildNavigationItemTrees(t *testing.T) {
	tests := []struct {
		name          string
		markdownMetas []models.MarkdownMeta
		expectedRoots int
	}{
		{
			name: "Normal tree structure",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway/SubFolder"},
				{Name: "File3", Path: "markdowns/Another-Folder"},
			},
			expectedRoots: 2, // "Gateway" and "Another-Folder"
		},
		{
			name: "Single-level path",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
			},
			expectedRoots: 1, // Only "Gateway"
		},
		{
			name: "Deeply nested path",
			markdownMetas: []models.MarkdownMeta{
				{Name: "DeepFile", Path: "markdowns/Gateway/SubFolder/SubSubFolder"},
			},
			expectedRoots: 1, // "Gateway" is root, "SubFolder" and "SubSubFolder" are children
		},
		{
			name: "Empty path",
			markdownMetas: []models.MarkdownMeta{
				{Name: "OrphanFile", Path: ""},
			},
			expectedRoots: 0, // No valid tree should be created
		},
		{
			name: "Invalid path",
			markdownMetas: []models.MarkdownMeta{
				{Name: "OrphanFile", Path: "helloInvalid"},
			},
			expectedRoots: 0, // No valid tree should be created
		},
		{
			name: "Invalid path parent",
			markdownMetas: []models.MarkdownMeta{
				{Name: "OrphanFile", Path: "not_markdown"},
			},
			expectedRoots: 0, // No valid tree should be created
		},
		{
			name: "Identical Href with different UUIDs",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway"},
			},
			expectedRoots: 1, // Should merge under the same Gateway parent
		},
		{
			name: "Multiple children under same parent",
			markdownMetas: []models.MarkdownMeta{
				{Name: "Child1", Path: "markdowns/Gateway"},
				{Name: "Child2", Path: "markdowns/Gateway"},
				{Name: "Child3", Path: "markdowns/Gateway"},
			},
			expectedRoots: 1, // "Gateway" as root with multiple children
		},
		{
			name: "One root w/o children, one root w/ multiple children under same parent",
			markdownMetas: []models.MarkdownMeta{
				{Name: "Gateway", Path: "markdowns"},
				{Name: "Child1", Path: "markdowns/Gateway"},
				{Name: "Child2", Path: "markdowns/Gateway"},
				{Name: "Child3", Path: "markdowns/Gateway"},
				{Name: "SingleRoot", Path: "markdowns"},
			},
			expectedRoots: 2, // "SingleRoot" as root w/o children & "Gateway" as root with multiple children
		},
	}

	c := collate.New(language.English)
	env := environment.Null()
	env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}
	s := markdowndoc.NavigationItemTreeService{Env: env, Collator: c}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run tree-building function
			result := s.BuildNavigationItemTrees(tt.markdownMetas)

			// Validate number of roots
			if len(result) != tt.expectedRoots {
				t.Errorf("test '%s': want %d roots, got %d", tt.name, tt.expectedRoots, len(result))
			}
		})
	}
}

func TestNavigationItemTopLevelOrder(t *testing.T) {
	tests := []struct {
		name          string
		markdownMetas []models.MarkdownMeta
		expectedOrder [][]string // slice of top-level elements (ordered)
	}{
		{
			name: "Proper roots order: 1",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway/SubFolder"},
				{Name: "File3", Path: "markdowns/Guidelines"},
				{Name: "File4", Path: "markdowns/Guidelines/SubFolder"},
			},
			expectedOrder: [][]string{
				{"Gateway", "Gateway"},       // should be at index 0
				{"Guidelines", "Guidelines"}, // should be at index 1
			},
		},
		{
			name: "Proper roots order: 2",
			markdownMetas: []models.MarkdownMeta{
				{Name: "my-file", Path: "markdowns/ABC"},
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway/SubFolder"},
				{Name: "File3", Path: "markdowns/Guidelines"},
				{Name: "File4", Path: "markdowns/Guidelines/SubFolder"},
				{Name: "File5", Path: "markdowns/A_Guidelines/SubFolder"},
				{Name: "File6", Path: "markdowns/Z_Guidelines/SubFolder"},
				{Name: "File7", Path: "markdowns/A-Guidelines/SubFolder"},
				{Name: "123456_large_number_first", Path: "markdowns"},
			},
			expectedOrder: [][]string{
				{"123456_large_number_first", "large number first"}, // should be at index 0
				{"A_Guidelines", "A Guidelines"},                    // should be at index 1
				{"A-Guidelines", "A-Guidelines"},                    // should be at index 2
				{"ABC", "ABC"},                                      // should be at index 3
				{"Gateway", "Gateway"},                              // should be at index 4
				{"Guidelines", "Guidelines"},                        // should be at index 5
				{"Z_Guidelines", "Z Guidelines"},                    // should be at index 6
			},
		},
		{
			name: "Proper roots order: with number prefix",
			markdownMetas: []models.MarkdownMeta{
				{Name: "my_file", Path: "markdowns/ABC"},
				{Name: "File1", Path: "markdowns/1_Gateway"},
				{Name: "File2", Path: "markdowns/1_Gateway/SubFolder"},
				{Name: "File3", Path: "markdowns/5_Guidelines"},
				{Name: "File4", Path: "markdowns/5_Guidelines/SubFolder"},
				{Name: "File5", Path: "markdowns/A_Guidelines/SubFolder"},
				{Name: "File6", Path: "markdowns/Z_Guidelines/SubFolder"},
			},
			expectedOrder: [][]string{
				{"Gateway", "Gateway"},           // should be at index 0
				{"Guidelines", "Guidelines"},     // should be at index 1
				{"A_Guidelines", "A Guidelines"}, // should be at index 2
				{"ABC", "ABC"},                   // should be at index 3
				{"Z_Guidelines", "Z Guidelines"}, // should be at index 4
			},
		},
	}

	c := collate.New(language.English)
	env := environment.Null()
	env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}
	s := markdowndoc.NavigationItemTreeService{Env: env, Collator: c}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			navigationTrees := s.BuildNavigationItemTrees(tt.markdownMetas)

			// verify that roots are sorted in the expected order
			for i, v := range tt.expectedOrder {
				root := navigationTrees[i]

				if v[0] != root.Href {
					t.Errorf("test '%s': Href: want %s, got %s", tt.name, v[0], navigationTrees[i].Href)
					return
				}

				if v[1] != root.Label {
					t.Errorf("test '%s': Label: want %s, got %s", tt.name, v[1], navigationTrees[i].Label)
					return
				}
			}
		})
	}
}

func TestNavigationItemHiddenTopLevelElements(t *testing.T) {
	tests := []struct {
		name          string
		markdownMetas []models.MarkdownMeta
		expectedRoots [][]string // slice of top-level elements (ordered and w/o hidden elements)
		hiddenRoots   []string   // slice of hidden top-level elements
	}{
		{
			name: "Hidden roots: 1",
			markdownMetas: []models.MarkdownMeta{
				{Name: "File1", Path: "markdowns/Gateway"},
				{Name: "File2", Path: "markdowns/Gateway/SubFolder"},
				{Name: "File3", Path: "markdowns/Guidelines"},
				{Name: "File4", Path: "markdowns/Guidelines/SubFolder"},
				{Name: "Hidden1", Path: "markdowns/.HiddenFolder/"},
				{Name: "Hidden2", Path: "markdowns/.HiddenFolder/"},
				{Name: "Hidden3", Path: "markdowns/.HiddenFolder/Subfolder"},
				{Name: ".HiddenMarkdown", Path: "markdowns"},
			},
			expectedRoots: [][]string{
				{"Gateway", "Gateway"},       // should be at index 0
				{"Guidelines", "Guidelines"}, // should be at index 1
			},
			hiddenRoots: []string{
				".HiddenFolder",
				".HiddenMarkdown",
			},
		},
	}

	c := collate.New(language.English)
	env := environment.Null()
	env.Logger = logging.DefaultLogger{Logger: zap.NewNop().Sugar()}
	s := markdowndoc.NavigationItemTreeService{Env: env, Collator: c}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			navigationTrees := s.BuildNavigationItemTrees(tt.markdownMetas)

			// verify that roots are sorted in the expected order
			for i, v := range tt.expectedRoots {
				if v[0] != navigationTrees[i].Href {
					t.Errorf("test '%s': Href: want %s, got %s", tt.name, v[0], navigationTrees[i].Href)
					return
				}

				if v[1] != navigationTrees[i].Label {
					t.Errorf("test '%s': Label: want %s, got %s", tt.name, v[1], navigationTrees[i].Label)
					return
				}
			}

			treesByHref := utils.SliceToMap(navigationTrees, func(root *markdowndoc.NavigationItem) string { return root.Href })

			for _, hidden := range tt.hiddenRoots {
				if _, ok := treesByHref[hidden]; ok {
					t.Errorf("test '%s': want no hidden root, found %s", tt.name, hidden)
					return
				}
			}
		})
	}
}

func TestTrigramSorensenDiceSimilarity_bounds(t *testing.T) {

	term := "hello"
	text := "hello this is a longer string with two hello"

	similarity := markdowndoc.TrigramSorensenDiceSimilarity(term, text)

	if similarity > 1.0 {
		t.Errorf("want similiarity to be less or equal to 1.0, got %f", similarity)
		return
	}

	if similarity < 0 {
		t.Errorf("want similiarity to be greater or equal to 0, got %f", similarity)
		return
	}
}

func TestTrigramSorensenDiceSimilarity_identical(t *testing.T) {
	want := 1.0

	term := "hello"
	text := "hello"

	got := markdowndoc.TrigramSorensenDiceSimilarity(term, text)

	if got != want {
		t.Errorf("want similiarity to be %f, got %f", want, got)
		return
	}
}

func TestTrigramSorensenDiceSimilarity_completely_different(t *testing.T) {
	var want float64

	term := "hello"
	text := ""

	got := markdowndoc.TrigramSorensenDiceSimilarity(term, text)

	if got != want {
		t.Errorf("want similiarity to be %f, got %f", want, got)
		return
	}
}

func TestTrigramSorensenDiceSimilarity(t *testing.T) {
	tests := []struct {
		A, B     string
		Expected float64
	}{
		{"hello", "hello", 1.0},
		{"hello", "world", 0.0},
		{"hello", "HELLO", 1.0},
		{"hello world", "world hello", 1.0},
		{"hello", "yellow", 0.307692},
		{"hello", "hell", 0.727273},
		{"hello world", "hello", 0.666667},
		{"hello", "helo", 0.727273},
		{"hello", "helloo", 0.769231},
		{"hello", "hella", 0.666667},
		{"hello", "help", 0.545455},
		{"hello", "halo", 0.363636},
		{"hello", "hell", 0.727273},
		{"hello", "hellish", 0.571429},
		{"hello", "helloo", 0.769231},
		{"hello", "helloooo", 0.7142857142857143},
		{"hello", "helloooooo", 0.7142857142857143},
		{"hello", "helloooooooo", 0.7142857142857143},
		{"hello", "helloooooooooo", 0.7142857142857143},
		{"hello", "helloooooooooooo", 0.7142857142857143},
	}

	for _, test := range tests {
		result := markdowndoc.TrigramSorensenDiceSimilarity(test.A, test.B)
		if math.Abs(result-test.Expected) > 1e-6 {
			t.Errorf("Similarity between %q and %q: want %f, got %f", test.A, test.B, test.Expected, result)
		}
	}
}

func BenchmarkTrigramSorensenDiceSimilarity_Asymmetric(b *testing.B) {
	small := "hello"
	large := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 10000) // ~560,000 characters

	b.Run("Small_vs_Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = markdowndoc.TrigramSorensenDiceSimilarity(small, large)
		}
	})

	b.Run("Large_vs_Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = markdowndoc.TrigramSorensenDiceSimilarity(large, small)
		}
	})
}

func TestTransformToUniqueTrigrams_emptyString_map(t *testing.T) {
	got := markdowndoc.TransformToUniqueTrigrams("")
	want := []string{}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
		return
	}
}

func TestTransformToUniqueTrigrams_hasPadding_map(t *testing.T) {
	text := "hello this is a longer string with two hello"
	got := markdowndoc.TransformToUniqueTrigrams(text)

	wantPaddingPrefix := "  "
	wantPaddingSuffix := " "

	first := got[0]
	if !strings.HasPrefix(first, wantPaddingPrefix) {
		t.Errorf("want first element to have prefix '%s', got '%s'", wantPaddingPrefix, first)
		return
	}

	last := got[len(got)-1]
	if !strings.HasSuffix(last, wantPaddingSuffix) {
		t.Errorf("want last element to have suffix '%s', got '%s'", wantPaddingSuffix, last)
		return
	}
}

func TestTransformToUniqueTrigrams_map(t *testing.T) {
	text := "hello this is a longer string with two hello"

	// 1 there's always one trigram because of padding
	// 2 aside from the initial trigram, we need to shift left n times
	//   with n equal to the count of character in the string (=> len(a))
	wantTrigrams := []string{"  a", "  h", "  i", "  l", "  s", "  t", "  w", " a ", " he", " is", " lo", " st", " th", " tw", " wi", "ell", "er ", "ger", "hel", "his", "ing", "is ", "ith", "llo", "lo ", "lon", "ng ", "nge", "ong", "rin", "str", "th ", "thi", "tri", "two", "wit", "wo "}
	wantNumOfTrigrams := len(wantTrigrams)

	got := markdowndoc.TransformToUniqueTrigrams(text)
	gotMap := make(map[string]string, len(got))
	for _, trigram := range got {
		gotMap[trigram] = trigram
	}

	if wantNumOfTrigrams != len(got) {
		t.Errorf("want '%d' number of trigrams, got '%d'", wantNumOfTrigrams, len(got))
		return
	}

	// this check makes it easier to identify missing trigrams than cmp.Diff,
	// especially if trigrams are unsorted
	for _, want := range wantTrigrams {
		if _, ok := gotMap[want]; !ok {
			t.Errorf("want '%s' trigrams, got none", want)
			return
		}
	}

	if !cmp.Equal(wantTrigrams, got) {
		t.Error(cmp.Diff(wantTrigrams, got))
		return
	}
}

func TestTransformToUniqueTrigrams_fuzzy_map(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "hello",
			expected: []string{"  h", " he", "ell", "hel", "llo", "lo "},
		},
		{
			input:    "hello hello",
			expected: []string{"  h", " he", "ell", "hel", "llo", "lo "},
		},
		{
			input:    "hello-world",
			expected: []string{"  h", "  w", " he", " wo", "ell", "hel", "ld ", "llo", "lo ", "orl", "rld", "wor"},
		},
		{
			input:    "Hello HELLO",
			expected: []string{"  h", " he", "ell", "hel", "llo", "lo "},
		},
		{
			input:    "hello   world",
			expected: []string{"  h", "  w", " he", " wo", "ell", "hel", "ld ", "llo", "lo ", "orl", "rld", "wor"},
		},
		{
			input:    "a b c",
			expected: []string{"  a", "  b", "  c", " a ", " b ", " c "},
		},
		{
			input:    "this is a longer sentence with multiple words",
			expected: []string{"  a", "  i", "  l", "  m", "  s", "  t", "  w", " a ", " is", " lo", " mu", " se", " th", " wi", " wo", "ce ", "ds ", "enc", "ent", "er ", "ger", "his", "ipl", "is ", "ith", "le ", "lon", "lti", "mul", "nce", "nge", "nte", "ong", "ord", "ple", "rds", "sen", "ten", "th ", "thi", "tip", "ult", "wit", "wor"},
		},
		{
			input:    "hello  hello   hello",
			expected: []string{"  h", " he", "ell", "hel", "llo", "lo "},
		},
	}

	for _, test := range tests {
		result := markdowndoc.TransformToUniqueTrigrams(test.input)
		if test.expected != nil && !reflect.DeepEqual(result, test.expected) {
			t.Errorf("For input '%s', want %v, got %v", test.input, test.expected, result)
		}
	}
}

func BenchmarkTransformToUniqueTrigrams_Short_map(b *testing.B) {
	input := "hello world"
	for i := 0; i < b.N; i++ {
		_ = markdowndoc.TransformToUniqueTrigrams(input)
	}
}

func BenchmarkTransformToUniqueTrigrams_Medium_map(b *testing.B) {
	input := "this is a medium length string with several words and some repetition like hello hello hello"
	for i := 0; i < b.N; i++ {
		_ = markdowndoc.TransformToUniqueTrigrams(input)
	}
}

func BenchmarkTransformToUniqueTrigrams_Long_map(b *testing.B) {
	input := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. " +
		"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur."
	for i := 0; i < b.N; i++ {
		_ = markdowndoc.TransformToUniqueTrigrams(input)
	}
}

func BenchmarkTransformToUniqueTrigrams_RepeatedWords_map(b *testing.B) {
	input := "hello hello hello hello hello hello hello hello hello hello"
	for i := 0; i < b.N; i++ {
		_ = markdowndoc.TransformToUniqueTrigrams(input)
	}
}

func BenchmarkTransformToUniqueTrigrams_Large_map(b *testing.B) {
	// Generate a large input string by repeating a sentence many times
	// ~570,000 characters => ~190 A4 pages
	base := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
	repeatCount := 10000
	largeInput := strings.Repeat(base, repeatCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = markdowndoc.TransformToUniqueTrigrams(largeInput)
	}
}
