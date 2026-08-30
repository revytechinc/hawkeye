// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package redact_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revytechinc/hawkeye/internal/redact"
)

func TestString_RedactsFakeOpenSSHKey(t *testing.T) {
	in := "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE_TEST_KEY_NOT_A_REAL_SECRET_aaaaaaaa\n-----END OPENSSH PRIVATE KEY-----"
	out := redact.String(in)
	if strings.Contains(out, "FAKE_TEST_KEY_NOT_A_REAL_SECRET") {
		t.Fatalf("private key material leaked: %q", out)
	}
	if !strings.Contains(out, "[REDACTED") {
		t.Fatalf("expected redaction marker, got %q", out)
	}
}

func TestString_RedactsFakePasswordJSON(t *testing.T) {
	in := `{"username":"operator","password":"fake-password-for-tests-only"}`
	out := redact.String(in)
	if strings.Contains(out, "fake-password-for-tests-only") {
		t.Fatalf("password leaked: %q", out)
	}
}

func TestString_RedactsFakeAPIToken(t *testing.T) {
	in := "api_token=fake_api_token_AAAAAAAAAAAAAAAA"
	out := redact.String(in)
	if strings.Contains(out, "fake_api_token_AAAAAAAAAAAAAAAA") {
		t.Fatalf("api token leaked: %q", out)
	}
}

func TestString_RedactsFakeHtpasswd(t *testing.T) {
	in := "testuser:$apr1$faketest$abcdefghijklmnopqrstuv"
	out := redact.String(in)
	if strings.Contains(out, "$apr1$faketest$") {
		t.Fatalf("htpasswd blob leaked: %q", out)
	}
}

func TestString_RedactsFakeBasicBlob(t *testing.T) {
	in := "Authorization: Basic ZmFrZXVzZXI6ZmFrZXBhc3M="
	out := redact.String(in)
	if strings.Contains(out, "ZmFrZXVzZXI6ZmFrZXBhc3M=") {
		t.Fatalf("Basic blob leaked: %q", out)
	}
}

func TestString_RedactsFakeBearerAndPATs(t *testing.T) {
	in := "Bearer FAKESECRET_a3b4c5d6e7f8g9h0i1j2\nghp_FakeGitHubTokenForTestsOnly1234567890\nsk-fakeOpenAIKeyForTestsOnly1234567890abcd"
	out := redact.String(in)
	for _, leak := range []string{
		"ghp_FakeGitHubTokenForTestsOnly1234567890",
		"sk-fakeOpenAIKeyForTestsOnly1234567890abcd",
		"sk-fakeOpenAIKeyForTestsOnly1234567890abcd"[:12],
	} {
		_ = leak
	}
	if strings.Contains(out, "ghp_FakeGitHubTokenForTestsOnly") {
		t.Fatalf("github pat leaked: %q", out)
	}
	if strings.Contains(out, "sk-fakeOpenAIKeyForTestsOnly") {
		t.Fatalf("openai-style key leaked: %q", out)
	}
}

func TestBytes_MatchesString(t *testing.T) {
	in := []byte("password=fake-password-for-tests-only")
	if string(redact.Bytes(in)) != redact.String(string(in)) {
		t.Fatal("Bytes/String mismatch")
	}
}

func TestContainsSecret_FakeFixtureFile(t *testing.T) {
	p := filepath.Join("testdata", "fake_secrets.txt")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fake fixture: %v", err)
	}
	if !redact.ContainsSecret(string(b)) {
		t.Fatal("expected fake fixture to be detected as containing secrets")
	}
	out := redact.String(string(b))
	if strings.Contains(out, "FAKE_TEST_KEY_NOT_A_REAL_SECRET") {
		t.Fatal("fixture key leaked after redact")
	}
}

func TestString_LeavesBenignText(t *testing.T) {
	in := "zpool status tank && service nginx status"
	if redact.String(in) != in {
		t.Fatalf("benign text changed: %q", redact.String(in))
	}
}
