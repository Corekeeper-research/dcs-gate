package main

import "regexp"

// redactionPatterns are applied to every chunk emitted via the SSE stream
// endpoint before the chunk is sent over the wire. The intent is to prevent
// the judge model from leaking secrets, paths, or credentials that may have
// somehow ended up in its context window or its training data and be
// surfaced inside its chain of thought.
//
// Policy: conservative. We prefer over-redaction (a false positive replaces
// innocent text with [REDACTED]) over under-redaction (a false negative
// leaks a real secret). Patterns target well-known token shapes (OpenAI /
// AWS / Google / GitHub / Slack) and absolute paths that would expose the
// host filesystem layout.
//
// The list is intentionally narrow: we do NOT include a generic "40+
// alphanumeric chars" pattern because that catches base64 fragments,
// embeddings serialised as strings, model identifiers, and many other
// innocent strings, producing too many false positives.
var redactionPatterns = []*regexp.Regexp{
	// API keys with well-known prefixes (high precision, low false positive).
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),       // OpenAI / Anthropic legacy
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),          // AWS access key id
	regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),    // Google API key
	regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),     // Google OAuth access token
	regexp.MustCompile(`gh[ps]_[0-9A-Za-z]{36,}`),   // GitHub personal access token
	regexp.MustCompile(`xox[baprs]-[0-9A-Za-z\-]+`), // Slack tokens
	// Absolute paths that expose host filesystem layout.
	regexp.MustCompile(`/home/[^/\s]+/[^\s]*`),
	regexp.MustCompile(`/kaggle/working/[^\s]*`),
	regexp.MustCompile(`/root/[^\s]*`),
	regexp.MustCompile(`/var/lib/[^\s]*`),
	// Plain-text credential assignments. (?i) = case-insensitive.
	regexp.MustCompile(`(?i)(password|secret|api[_-]?key|access[_-]?token)\s*[:=]\s*\S+`),
}

// sanitizeChunk redacts secret-shaped substrings from a chunk of model
// output before it is forwarded to the SSE client. Returns the sanitized
// chunk. Multiple patterns may match a single chunk; each match is
// independently replaced with the literal string "[REDACTED]".
func sanitizeChunk(chunk string) string {
	out := chunk
	for _, p := range redactionPatterns {
		out = p.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
