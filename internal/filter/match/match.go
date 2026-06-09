package match

import (
	"path"
	"strings"
)

type Type string

const (
	Is       Type = "is"
	Contains Type = "contains"
	Matches  Type = "matches"
)

func Compare(matchType Type, haystack, needle string) bool {
	switch matchType {
	case Is:
		return strings.EqualFold(haystack, needle)
	case Contains:
		return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
	case Matches:
		ok, err := path.Match(strings.ToLower(needle), strings.ToLower(haystack))
		return err == nil && ok
	default:
		return false
	}
}

func Any(matchType Type, haystacks []string, needles []string) bool {
	for _, haystack := range haystacks {
		for _, needle := range needles {
			if Compare(matchType, haystack, needle) {
				return true
			}
		}
	}
	return false
}
