package api

import (
	"testing"
)

func scopeTestStrPtr(s string) *string { return &s }

// TestScopeRuleInputToRuleHeaderValue is a regression test for hetty#142: the
// header value regexp must be compiled from the Value pattern, not the Key
// pattern (which silently discarded the user-supplied value matcher).
func TestScopeRuleInputToRuleHeaderValue(t *testing.T) {
	t.Parallel()

	rule, err := scopeRuleInputToRule(ScopeRuleInput{
		Header: &ScopeHeaderInput{
			Key:   scopeTestStrPtr("X-Custom"),
			Value: scopeTestStrPtr("secret"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.Header.Value == nil {
		t.Fatal("header value regexp was not compiled")
	}

	if !rule.Header.Value.MatchString("secret") {
		t.Errorf("header value regexp should match %q (compiled from the wrong field?)", "secret")
	}

	if rule.Header.Value.MatchString("public") {
		t.Errorf("header value regexp should not match %q", "public")
	}

	if rule.Header.Key == nil || !rule.Header.Key.MatchString("X-Custom") {
		t.Error("header key regexp should match \"X-Custom\"")
	}
}
