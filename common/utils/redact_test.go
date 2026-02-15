package utils

import "testing"

func TestRedactKeyValues(t *testing.T) {
	in := "password=abc123&token=xyz&secret=foo&signature=bar"
	out := Redact(in)
	if out == in {
		t.Fatalf("expected redacted output, got same: %s", out)
	}
	if !containsAll(out, []string{"password=****", "token=****", "secret=****", "signature=****"}) {
		t.Fatalf("redaction failed: %s", out)
	}
}

func TestRedactAuthorization(t *testing.T) {
	in := "Authorization: Bearer verysecrettoken123"
	out := Redact(in)
	if out == in || out == "" {
		t.Fatalf("expected redacted auth, got: %s", out)
	}
	if out != "Authorization: Bearer ****" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRedactEmail(t *testing.T) {
	in := "useremail@example.com"
	out := Redact(in)
	if out == in {
		t.Fatalf("expected redacted email, got: %s", out)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
