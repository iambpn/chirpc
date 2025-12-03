package rpc

import (
	"strings"
	"testing"
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

// testVerifyTsTypes normalizes whitespace and compares TypeScript output strings.
func testVerifyTsTypes(t *testing.T, types string, expectedTypes string) {
	t.Helper()

	replacer := strings.NewReplacer("\n", " ", "\t", "")

	types = strings.TrimSpace(replacer.Replace(types))
	expectedTypes = strings.TrimSpace(replacer.Replace(expectedTypes))

	if types != expectedTypes {
		t.Fatalf("expected output \n%s\n, got \n%s\n", expectedTypes, types)
	}
}
