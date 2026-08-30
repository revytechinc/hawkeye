// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package redact

import "regexp"

type rule struct {
	name string
	re   *regexp.Regexp
	repl string
}

var rules = []rule{
	{
		name: "openssh_key",
		re:   regexp.MustCompile(`(?s)-----BEGIN OPENSSH PRIVATE KEY-----.*?-----END OPENSSH PRIVATE KEY-----`),
		repl: "[REDACTED:openssh_key]",
	},
	{
		name: "private_key",
		re:   regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`),
		repl: "[REDACTED:private_key]",
	},
	{
		name: "htpasswd",
		re:   regexp.MustCompile(`(?m)[A-Za-z0-9._-]+:\$(?:apr1|2[aby]|6|5|1)\$\S+`),
		repl: "[REDACTED:htpasswd]",
	},
	{
		name: "basic_blob",
		re:   regexp.MustCompile(`(?i)(?:Authorization:\s*)?Basic\s+[A-Za-z0-9+/=_-]{8,}`),
		repl: "[REDACTED:basic]",
	},
	{
		name: "bearer",
		re:   regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-+=/]{8,}`),
		repl: "[REDACTED:bearer]",
	},
	{
		name: "github_pat",
		re:   regexp.MustCompile(`ghp_[A-Za-z0-9_]{20,}`),
		repl: "[REDACTED:github_pat]",
	},
	{
		name: "github_fine",
		re:   regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
		repl: "[REDACTED:github_pat]",
	},
	{
		name: "openai_sk",
		re:   regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),
		repl: "[REDACTED:api_token]",
	},
	{
		name: "password_json",
		re:   regexp.MustCompile(`(?i)("password"\s*:\s*")[^"]*(")`),
		repl: `${1}[REDACTED]${2}`,
	},
	{
		name: "password_kv",
		re:   regexp.MustCompile(`(?i)(password\s*[=:]\s*)\S+`),
		repl: `${1}[REDACTED]`,
	},
	{
		name: "api_token",
		re:   regexp.MustCompile(`(?i)((?:api[_-]?token|api[_-]?key|secret[_-]?key|access[_-]?token)\s*[=:]\s*)\S+`),
		repl: `${1}[REDACTED]`,
	},
}

// String returns s with SSH keys, passwords, tokens, htpasswd, and Basic blobs removed.
// Call this before any LLM or MCP payload. Tests must use FAKE fixtures only.
func String(s string) string {
	out := s
	for _, r := range rules {
		out = r.re.ReplaceAllString(out, r.repl)
	}
	return out
}

// Bytes redacts a byte slice.
func Bytes(b []byte) []byte {
	return []byte(String(string(b)))
}

// ContainsSecret reports whether redaction would change s.
func ContainsSecret(s string) bool {
	return String(s) != s
}
