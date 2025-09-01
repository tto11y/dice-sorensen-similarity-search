package utils

import (
	"math"
	"unicode"
)

// SliceToMap creates a map from a slice, using the provided key function to determine the map keys.
// If the key function returns a non-unique value for two or more elements, the resulting map will only contain the last element for that key.

// SliceToMap transforms a slice into a map by applying a key function to each element.
//
// The function takes a slice of type T and a key extraction function that returns a comparable key of type K.
// Each element from the slice becomes a value in the resulting map, indexed by the key derived from the element.
//
// If multiple elements produce the same key, the last one encountered in the slice will overwrite previous ones.
//
// param s the input slice to convert
// param key a function that extracts a comparable key from each element
// return a map[K]T representing the slice keyed by the extracted value
func SliceToMap[T any, K comparable](s []T, key func(T) K) map[K]T {
	m := make(map[K]T)

	for _, v := range s {
		m[key(v)] = v
	}

	return m
}

// ToSnakeCase converts a CamelCase string to snake_case.
// It inserts underscores before uppercase letters (except the first one)
// and converts all letters to lowercase.
//
// Examples:
//
//	ToSnakeCase("CamelCase")    => "camel_case"
//	ToSnakeCase("HTTPRequest")  => "http_request"
//	ToSnakeCase("UserID")       => "user_id"
func ToSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if unicode.IsUpper(r) {
			// Add underscore if:
			// - not the first character
			// - previous character is lower OR next character is lower (end of acronym)
			//
			// e.g., HTTPRequest will be converted to http_request
			if i > 0 && (unicode.IsLower(rune(str[i-1])) || (i+1 < len(str) && unicode.IsLower(rune(str[i+1])))) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
			continue
		}

		result = append(result, r)
	}

	return string(result)
}

// CalculateTotalPages computes the total number of pages required to display all elements,
// given the total number of matching elements (`matchCount`) and the number of elements per page (`pageSize`).
//
// It performs a ceiling division to ensure that any remaining elements that don't fill a full page
// still count as an additional page.
//
// If `pageSize` is zero or negative, the function returns 0 to avoid division by zero.
//
// Parameters:
//   - matchCount: the total number of elements to paginate
//   - pageSize: the number of elements per page
//
// Returns:
//   - int: the total number of pages needed
func CalculateTotalPages(matchCount, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}

	exactPageSize := float64(matchCount) / float64(pageSize)
	return int(math.Ceil(exactPageSize))
}
