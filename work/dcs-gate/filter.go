package main

import "regexp"

// redactionPatterns contains conservative regular expressions that match
// strings commonly found in chain-of-thought traces emitted by reasoning
// judges. Every chunk emitted via SSE on /auth/stream is passed through
// sanitizeChunk before reaching the network. The policy is intentionally
// over-conservative: replacing a piece of innocent text with [REDACTED] is
// preferred to leaking a real secret. Order matters — the most specific
// patterns run first so we redact "sk-…" as a whole rather than letting the
// generic long-token rule capture only part of it.
var redactionPatterns = []*regexp.Regexp{
	// Provider-specific API key prefixes (high signal).
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{20,}`),       // OpenAI
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),             // AWS Access Key ID
	regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),       // Google API key
	regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),        // Google OAuth refresh
	regexp.MustCompile(`gh[ps]_[0-9A-Za-z]{36,}`),      // GitHub PAT (classic + new)
	regexp.MustCompile(`xox[baprs]-[0-9A-Za-z\-]+`),    // Slack tokens
	regexp.MustCompile(`Bearer\s+[A-Za-z0-9_\-\.]+`),   // HTTP auth header
	// Plain-text credential assignments (high recall).
	regexp.MustCompile(`(?i)(password|secret|token|api[_\-]?key|access[_\-]?key|auth[_\-]?token)\s*[:=]\s*\S+`),
	// Absolute filesystem paths likely to contain user identifiers.
	regexp.MustCompile(`/home/[^\s/]+(?:/[^\s]*)?`),
	regexp.MustCompile(`/kaggle/working(?:/[^\s]*)?`),
	regexp.MustCompile(`/root(?:/[^\s]*)?`),
	regexp.MustCompile(`/var/lib(?:/[^\s]*)?`),
	// Generic long-token catch-all. Runs LAST so the prefixed rules above
	// claim well-known keys first. 40+ alnum chars with optional dashes is
	// the signature of most opaque tokens.
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{40,}\b`),
}

// sanitizeChunk applies every redactionPattern to the input, replacing each
// match with the literal token [REDACTED]. The function is idempotent (a
// chunk already redacted yields the same output again) and side-effect free
// (no logging, no allocation beyond the result string). Callers should treat
// the return value as the only safe-to-emit form of the chunk.
func sanitizeChunk(chunk string) string {
	out := chunk
	for _, p := range redactionPatterns {
		out = p.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
