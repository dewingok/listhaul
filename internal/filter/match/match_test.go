package match

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name      string
		matchType Type
		haystack  string
		needle    string
		want      bool
	}{
		{"is equal", Is, "Hello", "hello", true},
		{"is different", Is, "Hello", "world", false},
		{"contains", Contains, "Weekly Newsletter", "newsletter", true},
		{"matches glob", Matches, "Big Sale Today", "*sale*", true},
		{"matches miss", Matches, "Big Sale Today", "*viagra*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.matchType, tt.haystack, tt.needle); got != tt.want {
				t.Fatalf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAny(t *testing.T) {
	if !Any(Contains, []string{"foo", "bar baz"}, []string{"baz"}) {
		t.Fatal("expected match across haystacks")
	}
}
