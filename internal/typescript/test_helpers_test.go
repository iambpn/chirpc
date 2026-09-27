package typescript

import (
	"strings"
	"testing"
	"unicode"
)

// Generic HTTP response used in tests to model handler return types.
type testHttpResponse[T any] struct {
	StatusCode int
	Body       T
	Headers    map[string]string
}

// Sample nested types used across tests.
type testAddress struct {
	Line1 string
	Zip   int
}

type testUserProfile struct {
	Name    string
	Primary testAddress
}

type testTeamPayload struct {
	Owner   testUserProfile
	Members []testUserProfile
}

type testCreateReq struct {
	Name   string
	TagIds []int
}

type testSearchQ struct {
	Filter string
	Limit  int
}

type testOptionalQ struct {
	Filter string `json:"filter,omitempty"`
	Limit  int    `tsOptional:"true"`
}

// testCompact removes all whitespace so TypeScript output can be compared regardless of formatting.
func testCompact(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}

// testVerifyTsTypes normalizes whitespace and compares TypeScript output strings.
func testVerifyTsTypes(t *testing.T, types string, expectedTypes string) {
	t.Helper()

	if testCompact(types) != testCompact(expectedTypes) {
		t.Fatalf("expected output \n%s\n, got \n%s\n", expectedTypes, types)
	}
}
