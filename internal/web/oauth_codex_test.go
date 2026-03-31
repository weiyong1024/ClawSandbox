package web

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		t.Fatalf("generatePKCE: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatal("verifier or challenge is empty")
	}
	if strings.ContainsAny(verifier, "+/=") {
		t.Fatalf("verifier contains non-base64url chars: %s", verifier)
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Fatalf("challenge contains non-base64url chars: %s", challenge)
	}
	if verifier == challenge {
		t.Fatal("verifier and challenge should differ")
	}
}

func TestGeneratePKCEUniqueness(t *testing.T) {
	v1, _, _ := generatePKCE()
	v2, _, _ := generatePKCE()
	if v1 == v2 {
		t.Fatal("two PKCE verifiers should not be identical")
	}
}

func TestBuildAndParseState(t *testing.T) {
	nonce := "abc123def456"
	origin := "http://localhost:8081"
	state := buildState(nonce, origin)

	gotNonce, gotOrigin, ok := parseState(state)
	if !ok {
		t.Fatalf("parseState failed for %q", state)
	}
	if gotNonce != nonce {
		t.Fatalf("nonce: got %q, want %q", gotNonce, nonce)
	}
	if gotOrigin != origin {
		t.Fatalf("origin: got %q, want %q", gotOrigin, origin)
	}
}

func TestParseStateInvalid(t *testing.T) {
	cases := []string{"", "nosep", ".leadingdot", "trailingdot."}
	for _, s := range cases {
		if _, _, ok := parseState(s); ok {
			t.Fatalf("parseState should fail for %q", s)
		}
	}
}

func TestDecodeJWTAccountID(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := map[string]any{
		"sub":  "user-123",
		"name": "Test User",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_abc123",
			"user_id":            "user-123",
		},
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	fakeJWT := header + "." + payloadB64 + ".fake-signature"

	accountID := decodeJWTAccountID(fakeJWT)
	if accountID != "acct_abc123" {
		t.Fatalf("expected acct_abc123, got %q", accountID)
	}
}

func TestDecodeJWTAccountIDMissingClaim(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload := map[string]any{"sub": "user-123"}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	fakeJWT := header + "." + payloadB64 + ".sig"

	if id := decodeJWTAccountID(fakeJWT); id != "" {
		t.Fatalf("expected empty for missing claim, got %q", id)
	}
}

func TestDecodeJWTAccountIDInvalidToken(t *testing.T) {
	if id := decodeJWTAccountID("not-a-jwt"); id != "" {
		t.Fatalf("expected empty for invalid token, got %q", id)
	}
	if id := decodeJWTAccountID(""); id != "" {
		t.Fatalf("expected empty for empty token, got %q", id)
	}
}

func TestBuildAuthURL(t *testing.T) {
	u := buildAuthURL("nonce123.http://localhost:8080", "test-challenge")
	if !strings.Contains(u, "auth.openai.com") {
		t.Fatalf("URL missing auth.openai.com: %s", u)
	}
	if !strings.Contains(u, "code_challenge=test-challenge") {
		t.Fatalf("URL missing code_challenge: %s", u)
	}
	if !strings.Contains(u, "code_challenge_method=S256") {
		t.Fatalf("URL missing S256 method: %s", u)
	}
	if !strings.Contains(u, codexClientID) {
		t.Fatalf("URL missing client_id: %s", u)
	}
}

func TestCallbackHTML(t *testing.T) {
	html := callbackHTML("Test Title", "Test message", false)
	if !strings.Contains(html, "Test Title") {
		t.Fatal("missing title")
	}
	if !strings.Contains(html, "#22c55e") {
		t.Fatal("expected green for success")
	}

	errHTML := callbackHTML("Error", "Fail", true)
	if !strings.Contains(errHTML, "#ef4444") {
		t.Fatal("expected red for error")
	}
}
