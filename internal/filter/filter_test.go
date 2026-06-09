package filter

import (
	"net/textproto"
	"testing"

	"github.com/dewingok/listhaul/internal/config"
)

func TestEvaluateHeader(t *testing.T) {
	headers := textproto.MIMEHeader{
		"List-Id": {"<newsletter.lists.example.com>"},
	}

	test := config.Test{
		Header: &config.HeaderTest{
			Names:  []string{"List-Id"},
			Match:  "contains",
			Values: []string{"lists.example.com"},
		},
	}
	if !EvaluateTest(test, headers) {
		t.Fatal("expected header match")
	}
}

func TestEvaluateAnyOf(t *testing.T) {
	headers := textproto.MIMEHeader{
		"Subject": {"Limited time sale"},
	}

	test := config.Test{
		AnyOf: []config.Test{
			{
				Header: &config.HeaderTest{
					Names:  []string{"Subject"},
					Match:  "matches",
					Values: []string{"*sale*"},
				},
			},
			{
				Not: &config.Test{
					Exists: []string{"List-Id"},
				},
			},
		},
	}
	if !EvaluateTest(test, headers) {
		t.Fatal("expected anyof match")
	}
}

func TestEvaluateRulesStop(t *testing.T) {
	headers := textproto.MIMEHeader{
		"List-Id": {"lists.example.com"},
	}

	rules := []config.Rule{
		{
			Name: "newsletters",
			If: config.Test{
				Header: &config.HeaderTest{
					Names:  []string{"List-Id"},
					Match:  "contains",
					Values: []string{"lists.example.com"},
				},
			},
			Then: []config.ThenAction{
				{Action: "fileinto", Folder: "INBOX/Newsletters"},
				{Action: "stop"},
			},
		},
		{
			Name: "should-not-run",
			If: config.Test{
				Header: &config.HeaderTest{
					Names:  []string{"Subject"},
					Match:  "contains",
					Values: []string{"anything"},
				},
			},
			Then: []config.ThenAction{{Action: "discard"}},
		},
	}

	result, matched := EvaluateRules(rules, headers)
	if !matched {
		t.Fatal("expected rule match")
	}
	if result.RuleName != "newsletters" {
		t.Fatalf("rule = %q, want newsletters", result.RuleName)
	}
	if !result.Stop {
		t.Fatal("expected stop")
	}
}
