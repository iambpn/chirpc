package chirpc

import "testing"

func TestParseURLSlugSingle(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/users/{id}", []string{"id"}},
		{"{one}", []string{"one"}},
		{"/no/slugs/here", []string{}},
		{"", []string{}},
	}

	for _, c := range cases {
		got := parseURLSlug(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("parseURLSlug(%q) len = %d, want %d (%v)", c.in, len(got), len(c.want), got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("parseURLSlug(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestParseURLSlugMultiple(t *testing.T) {
	got := parseURLSlug("/{user}/{id}")
	if len(got) != 2 || got[0] != "user" || got[1] != "id" {
		t.Fatalf("parseURLSlug multi failed: got %v, want [user id]", got)
	}
}
