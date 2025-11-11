package chirpc

/*
Helper functions for handling URL parsing.
*/

// parseURLSlug extracts URL path parameter names enclosed in braces from the
// given url string.
//
// It scans the input and collects substrings that appear between a first-level
// opening '{' and its corresponding '}' character. Nested brace groups are
// handled by counting opening braces and only capturing text when the
// corresponding first-level closing brace is reached. The function returns a
// slice of slug names in the order they appear in the URL.
func parseURLSlug(url string) []string {
	slugs := []string{}

	slug := ""
	bracketOpenCount := 0
	for _, part := range url {
		if part == '{' {
			bracketOpenCount++
			continue
		}

		if part == '}' && bracketOpenCount == 1 {
			bracketOpenCount--
			slugs = append(slugs, slug)
			slug = ""
			continue
		}

		if bracketOpenCount > 0 {
			slug += string(part)
		}
	}

	return slugs
}

func mergePaths(basePath, relativePath string) string {
	if basePath == "" {
		return relativePath
	}
	if relativePath == "" {
		return basePath
	}

	hasBaseSlash := basePath[len(basePath)-1] == '/'
	hasRelativeSlash := relativePath[0] == '/'

	switch {
	case hasBaseSlash && hasRelativeSlash:
		return basePath + relativePath[1:]
	case !hasBaseSlash && !hasRelativeSlash:
		return basePath + "/" + relativePath
	default:
		return basePath + relativePath
	}
}
