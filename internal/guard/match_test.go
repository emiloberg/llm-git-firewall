package guard

import "testing"

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// Exact match
		{"git push origin main", "git push origin main", true},
		{"git push origin main", "git push origin develop", false},

		// Wildcard at end — matches simple branch
		{"git push origin *", "git push origin feat/foo", true},

		// Wildcard at end — matches nested branch
		{"git push origin feat/*", "git push origin feat/sub/branch", true},

		// Wildcard at end — no match
		{"git push origin feat/*", "git push origin fix/bar", false},

		// Wildcard in middle
		{"git push * --force", "git push origin --force", true},
		{"git push * --force", "git push origin main", false},

		// Multiple wildcards
		{"git push * *", "git push origin feat/foo", true},

		// Empty input
		{"git push origin *", "", false},

		// --force anywhere (pattern: *--force*)
		{"*--force*", "git push --force origin feat/foo", true},
		{"*--force*", "git push origin --force feat/foo", true},
		{"*--force*", "git push origin feat/foo --force", true},
		{"*--force*", "git push origin feat/foo", false},
		// --force-with-lease caught by same pattern
		{"*--force*", "git push --force-with-lease origin feat/foo", true},
		{"*--force*", "git push origin feat/foo --force-with-lease", true},

		// -f at end (pattern: * -f)
		{"* -f", "git push origin feat/foo -f", true},
		{"* -f", "git push -f origin feat/foo", false},
		{"* -f", "git push origin feat/foo", false},

		// -f in middle (pattern: * -f *)
		{"* -f *", "git push -f origin feat/foo", true},
		{"* -f *", "git push origin -f feat/foo", true},
		{"* -f *", "git push origin feat/foo -f", false},
		{"* -f *", "git push origin feat/foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.input, func(t *testing.T) {
			got := MatchPattern(tt.pattern, tt.input)
			if got != tt.want {
				t.Errorf("MatchPattern(%q, %q) = %v, want %v", tt.pattern, tt.input, got, tt.want)
			}
		})
	}
}
