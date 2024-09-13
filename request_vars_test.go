package tupa

import (
	"testing"
)

func TestExtractParams(t *testing.T) {
	tests := []struct {
		pattern    string
		path       string
		expected   map[string]string
		shouldFail bool
	}{
		{
			pattern:    "/users/{id}",
			path:       "/users/123",
			expected:   map[string]string{"id": "123"},
			shouldFail: false,
		},
		{
			pattern:    "/users/{id}/books/{bookId}",
			path:       "/users/123/books/456",
			expected:   map[string]string{"id": "123", "bookId": "456"},
			shouldFail: false,
		},
		{
			pattern:    "/users/{id}/books/{bookId}",
			path:       "/users/123/books/",
			expected:   map[string]string{"id": "123", "bookId": ""},
			shouldFail: false,
		},
		{
			pattern:    "/users/{id}/books/{bookId}",
			path:       "/users/123/books",
			expected:   map[string]string{"id": "123", "bookId": ""},
			shouldFail: true,
		},
		{
			pattern:    "/users/{id}",
			path:       "/users/",
			expected:   map[string]string{"id": ""},
			shouldFail: false,
		},
		{
			pattern:    "/users/{id}",
			path:       "/user/123",
			expected:   map[string]string{},
			shouldFail: true,
		},
		{
			pattern:    "/",
			path:       "/",
			expected:   map[string]string{},
			shouldFail: false,
		},
	}

	for _, test := range tests {
		t.Run(test.pattern+"_"+test.path, func(t *testing.T) {
			result := extractParams(test.pattern, test.path)
			if test.shouldFail {
				if len(result) != 0 {
					t.Errorf("expected failure, got %v", result)
				}
			} else {
				if !equal(result, test.expected) {
					t.Errorf("expected %v, got %v", test.expected, result)
				}
			}
		})
	}
}

func equal(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valueA := range a {
		if valueB, ok := b[key]; !ok || valueA != valueB {
			return false
		}
	}
	return true
}
