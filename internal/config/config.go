package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	DefaultPollInterval = 10 * time.Minute
	DefaultPollLookback = 60 * time.Minute
	DefaultMailbox      = "INBOX"
	DefaultConfigPath   = "./listhaul.toml"
)

var defaultEnabledActions = []string{"fileinto", "discard", "mark_read", "flag", "stop"}

type Config struct {
	Poll    PollConfig    `toml:"poll"`
	Account AccountConfig `toml:"account"`
	State   StateConfig   `toml:"state"`
	Actions ActionsConfig `toml:"actions"`
	Rules   []Rule        `toml:"rules"`
}

type PollConfig struct {
	Interval   Duration `toml:"interval"`
	Lookback   Duration `toml:"lookback"`
	UnseenOnly bool     `toml:"unseen_only"`
}

type AccountConfig struct {
	Host        string `toml:"host"`
	Port        int    `toml:"port"`
	Username    string `toml:"username"`
	PasswordEnv string `toml:"password_env"`
	Mailbox     string `toml:"mailbox"`
}

type StateConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

type ActionsConfig struct {
	Enabled []string `toml:"enabled"`
}

type Rule struct {
	Name string       `toml:"name"`
	If   Test         `toml:"if"`
	Then []ThenAction `toml:"then"`
}

type Test struct {
	Header *HeaderTest `toml:"header"`
	Exists []string    `toml:"exists"`
	Not    *Test       `toml:"not"`
	AnyOf  []Test      `toml:"anyof"`
	AllOf  []Test      `toml:"allof"`
}

type HeaderTest struct {
	Names  []string `toml:"names"`
	Match  string   `toml:"match"`
	Values []string `toml:"values"`
}

type ThenAction struct {
	Action string `toml:"action"`
	Folder string `toml:"folder"`
	Flag   string `toml:"flag"`
	Set    *bool  `toml:"set"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	d.Duration = parsed
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Poll.Interval.Duration == 0 {
		c.Poll.Interval.Duration = DefaultPollInterval
	}
	if c.Poll.Lookback.Duration == 0 {
		c.Poll.Lookback.Duration = DefaultPollLookback
	}
	if c.Account.Port == 0 {
		c.Account.Port = 993
	}
	if c.Account.Mailbox == "" {
		c.Account.Mailbox = DefaultMailbox
	}
	if len(c.Actions.Enabled) == 0 {
		c.Actions.Enabled = append([]string(nil), defaultEnabledActions...)
	}
	if c.State.Path == "" {
		c.State.Path = "~/.listhaul/state.json"
	}
}

func (c *Config) Validate() error {
	if c.Account.Host == "" {
		return fmt.Errorf("account.host is required")
	}
	if c.Account.Username == "" {
		return fmt.Errorf("account.username is required")
	}
	if c.Account.PasswordEnv == "" {
		return fmt.Errorf("account.password_env is required")
	}
	if c.Poll.Interval.Duration <= 0 {
		return fmt.Errorf("poll.interval must be positive")
	}
	if c.Poll.Lookback.Duration <= 0 {
		return fmt.Errorf("poll.lookback must be positive")
	}

	enabled := make(map[string]struct{}, len(c.Actions.Enabled))
	for _, action := range c.Actions.Enabled {
		action = strings.TrimSpace(action)
		if action == "" {
			return fmt.Errorf("actions.enabled contains empty entry")
		}
		enabled[action] = struct{}{}
	}

	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	for i, rule := range c.Rules {
		if err := validateTest(rule.If, fmt.Sprintf("rules[%d].if", i)); err != nil {
			return err
		}
		if len(rule.Then) == 0 {
			return fmt.Errorf("rules[%d] must have at least one then action", i)
		}
		for j, action := range rule.Then {
			if action.Action == "" {
				return fmt.Errorf("rules[%d].then[%d].action is required", i, j)
			}
			if _, ok := enabled[action.Action]; !ok {
				return fmt.Errorf("rules[%d].then[%d] action %q is not enabled in actions.enabled", i, j, action.Action)
			}
			if err := validateThenAction(action, i, j); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateTest(test Test, path string) error {
	kinds := 0
	if test.Header != nil {
		kinds++
		if err := validateHeaderTest(*test.Header, path+".header"); err != nil {
			return err
		}
	}
	if len(test.Exists) > 0 {
		kinds++
	}
	if test.Not != nil {
		kinds++
		if err := validateTest(*test.Not, path+".not"); err != nil {
			return err
		}
	}
	if len(test.AnyOf) > 0 {
		kinds++
		for i, child := range test.AnyOf {
			if err := validateTest(child, fmt.Sprintf("%s.anyof[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	if len(test.AllOf) > 0 {
		kinds++
		for i, child := range test.AllOf {
			if err := validateTest(child, fmt.Sprintf("%s.allof[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	if kinds == 0 {
		return fmt.Errorf("%s must define exactly one test type", path)
	}
	if kinds > 1 {
		return fmt.Errorf("%s must define only one test type", path)
	}
	return nil
}

func validateHeaderTest(test HeaderTest, path string) error {
	if len(test.Names) == 0 {
		return fmt.Errorf("%s.names is required", path)
	}
	if test.Match == "" {
		return fmt.Errorf("%s.match is required", path)
	}
	switch test.Match {
	case "is", "contains", "matches":
	default:
		return fmt.Errorf("%s.match must be is, contains, or matches", path)
	}
	if len(test.Values) == 0 {
		return fmt.Errorf("%s.values is required", path)
	}
	return nil
}

func validateThenAction(action ThenAction, ruleIdx, actionIdx int) error {
	path := fmt.Sprintf("rules[%d].then[%d]", ruleIdx, actionIdx)
	switch action.Action {
	case "fileinto":
		if action.Folder == "" {
			return fmt.Errorf("%s.folder is required for fileinto", path)
		}
	case "flag":
		if action.Flag == "" {
			return fmt.Errorf("%s.flag is required for flag action", path)
		}
		if action.Set == nil {
			return fmt.Errorf("%s.set is required for flag action", path)
		}
	case "discard", "mark_read", "stop":
	default:
		return fmt.Errorf("%s unknown action %q", path, action.Action)
	}
	return nil
}

func (c *Config) Password() (string, error) {
	value := os.Getenv(c.Account.PasswordEnv)
	if value == "" {
		return "", fmt.Errorf("environment variable %q is not set", c.Account.PasswordEnv)
	}
	return value, nil
}

func (c *Config) EnabledActions() map[string]struct{} {
	enabled := make(map[string]struct{}, len(c.Actions.Enabled))
	for _, action := range c.Actions.Enabled {
		enabled[action] = struct{}{}
	}
	return enabled
}
