package kiro

import "testing"

func TestResolveRedirectURI_DefaultsToNumericLoopback(t *testing.T) {
	got, err := ResolveRedirectURI("")
	if err != nil {
		t.Fatalf("ResolveRedirectURI returned error: %v", err)
	}
	if got != DefaultRedirectURI {
		t.Fatalf("redirect URI mismatch: got=%q want=%q", got, DefaultRedirectURI)
	}
}

func TestResolveRedirectURI_NormalizesLocalhost(t *testing.T) {
	got, err := ResolveRedirectURI("http://localhost:19877/oauth/callback")
	if err != nil {
		t.Fatalf("ResolveRedirectURI returned error: %v", err)
	}
	const want = "http://127.0.0.1:19877/oauth/callback"
	if got != want {
		t.Fatalf("redirect URI mismatch: got=%q want=%q", got, want)
	}
}

func TestResolveRedirectURI_RejectsNonLoopbackHost(t *testing.T) {
	_, err := ResolveRedirectURI("https://sub2api.example.com/oauth/callback")
	if err == nil {
		t.Fatal("expected error for non-loopback redirect URI")
	}
}
