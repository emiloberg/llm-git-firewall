package guard

import "strings"

// MatchPattern checks if input matches a glob pattern where * matches any
// string including /.
func MatchPattern(pattern, input string) bool {
	// Split pattern by * to get literal segments
	parts := strings.Split(pattern, "*")

	if len(parts) == 1 {
		// No wildcard — exact match
		return pattern == input
	}

	// Check prefix (before first *)
	if !strings.HasPrefix(input, parts[0]) {
		return false
	}

	// Check suffix (after last *)
	if !strings.HasSuffix(input, parts[len(parts)-1]) {
		return false
	}

	// Walk through middle segments in order
	remaining := input[len(parts[0]):]
	for _, part := range parts[1:] {
		idx := strings.Index(remaining, part)
		if idx < 0 {
			return false
		}
		remaining = remaining[idx+len(part):]
	}

	return true
}
