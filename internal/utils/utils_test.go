package utils_test

import (
	"dice-sorensen-similarity-search/internal/auth"
	"dice-sorensen-similarity-search/internal/bitbucket"
	"dice-sorensen-similarity-search/internal/constants"
	"dice-sorensen-similarity-search/internal/environment"
	"dice-sorensen-similarity-search/internal/logging"
	"dice-sorensen-similarity-search/internal/markdowndoc"
	"dice-sorensen-similarity-search/internal/utils"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestSliceToMap(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	users := []User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
		{ID: 3, Name: "Charlie"},
	}

	want := map[int]User{
		1: {ID: 1, Name: "Alice"},
		2: {ID: 2, Name: "Bob"},
		3: {ID: 3, Name: "Charlie"},
	}

	got := utils.SliceToMap(users, func(u User) int { return u.ID })

	if !cmp.Equal(got, want) {
		t.Errorf("SliceToMap mismatch\n got:  %#v\nwant: %#v", got, want)
		return
	}
}

func TestRegistry(t *testing.T) {
	controllerRegistry := make(map[int]any)

	bPtr := &bitbucket.Controller{}
	controllerRegistry[constants.Bitbucket] = bPtr
	var core zapcore.Core
	bPtr.Env = &environment.Env{Logger: logging.DefaultLogger{Logger: zap.New(core).Sugar()}}

	hPtr := &markdowndoc.Controller{}
	controllerRegistry[constants.MarkdownDoc] = hPtr

	aPtr := &auth.Controller{}
	controllerRegistry[constants.Auth] = aPtr

	if bPtr != controllerRegistry[constants.Bitbucket] {
		t.Errorf("Bitbucket controller registry mismatch")
		return
	}

	if hPtr != controllerRegistry[constants.MarkdownDoc] {
		t.Errorf("Markdown doc controller registry mismatch")
		return
	}

	if aPtr != controllerRegistry[constants.Auth] {
		t.Errorf("Auth controller registry mismatch")
		return
	}

}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CamelCase", "camel_case"},
		{"HTTPRequest", "http_request"},
		{"UserID", "user_id"},
		{"SimpleTest", "simple_test"},
		{"Already_snake_case", "already_snake_case"},
		{"lowercase", "lowercase"},
		{"", ""},
	}

	for _, test := range tests {
		got := utils.ToSnakeCase(test.input)

		if got != test.want {
			t.Errorf("ToSnakeCase(%q) = %q; want %q", test.input, got, test.want)
			return
		}
	}
}

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		matchCount int
		pageSize   int
		want       int
	}{
		{matchCount: 0, pageSize: 10, want: 0},
		{matchCount: 10, pageSize: 10, want: 1},
		{matchCount: 15, pageSize: 10, want: 2},
		{matchCount: 25, pageSize: 10, want: 3},
		{matchCount: 100, pageSize: 25, want: 4},
		{matchCount: 101, pageSize: 25, want: 5},
		{matchCount: 50, pageSize: 0, want: 0}, // edge case: division by zero
	}

	for _, tt := range tests {
		got := utils.CalculateTotalPages(tt.matchCount, tt.pageSize)

		if got != tt.want {
			t.Errorf("CalculateTotalPages(%d, %d) = %d; want %d", tt.matchCount, tt.pageSize, got, tt.want)
			return
		}
	}
}
