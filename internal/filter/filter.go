package filter

import (
	"net/textproto"

	"github.com/dewingok/listhaul/internal/config"
	"github.com/dewingok/listhaul/internal/filter/match"
)

func EvaluateTest(test config.Test, headers textproto.MIMEHeader) bool {
	switch {
	case test.Header != nil:
		return evaluateHeader(*test.Header, headers)
	case len(test.Exists) > 0:
		return evaluateExists(test.Exists, headers)
	case test.Not != nil:
		return !EvaluateTest(*test.Not, headers)
	case len(test.AnyOf) > 0:
		for _, child := range test.AnyOf {
			if EvaluateTest(child, headers) {
				return true
			}
		}
		return false
	case len(test.AllOf) > 0:
		for _, child := range test.AllOf {
			if !EvaluateTest(child, headers) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func evaluateHeader(test config.HeaderTest, headers textproto.MIMEHeader) bool {
	var values []string
	for _, name := range test.Names {
		values = append(values, headers.Values(name)...)
	}
	return match.Any(match.Type(test.Match), values, test.Values)
}

func evaluateExists(names []string, headers textproto.MIMEHeader) bool {
	for _, name := range names {
		if len(headers.Values(name)) > 0 {
			return true
		}
	}
	return false
}

type MatchResult struct {
	RuleName string
	Actions  []config.ThenAction
	Stop     bool
}

func EvaluateRules(rules []config.Rule, headers textproto.MIMEHeader) (MatchResult, bool) {
	for _, rule := range rules {
		if !EvaluateTest(rule.If, headers) {
			continue
		}

		result := MatchResult{RuleName: rule.Name}
		for _, action := range rule.Then {
			if action.Action == "stop" {
				result.Stop = true
				result.Actions = append(result.Actions, action)
				return result, true
			}
			result.Actions = append(result.Actions, action)
		}
		return result, true
	}
	return MatchResult{}, false
}
