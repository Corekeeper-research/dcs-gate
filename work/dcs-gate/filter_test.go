package main

import (
	"strings"
	"testing"
)

func TestSanitizeChunkRedactsCommonSecrets(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // expected output; if "" we just assert it contains [REDACTED]
	}{
		// 1. OpenAI key prefix.
		{
			name: "openai_key",
			in:   "calling openai sk-abc123def456ghi789jkl012XYZ now",
			want: "calling openai [REDACTED] now",
		},
		// 2. AWS access key.
		{
			name: "aws_access_key",
			in:   "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			// the credential assignment rule matches first and gobbles the
			// whole "AKIA…" suffix, so we only assert presence of REDACTED
			// and absence of the raw key.
			want: "",
		},
		// 3. Google API key.
		{
			name: "google_api_key",
			in:   "GOOGLE_KEY=AIzaSyA1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R",
			want: "",
		},
		// 4. Google OAuth refresh.
		{
			name: "google_oauth",
			in:   "refresh token ya29.A0AfH6SM_xxxxxx_yyyyyy",
			want: "refresh token [REDACTED]",
		},
		// 5. GitHub PAT.
		{
			name: "github_pat",
			in:   "git clone https://ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789@github.com/me/repo.git",
			want: "",
		},
		// 6. Slack token.
		{
			name: "slack_token",
			in:   "Slack: xoxb-FAKE-PLACEHOLDER-NOT-A-REAL-TOKEN",
			want: "",
		},
		// 7. Bearer header.
		{
			name: "bearer_header",
			in:   "Authorization: Bearer eyJhbGciOi.JhbGciOiJIUzI1NiJ9.signature",
			want: "Authorization: [REDACTED]",
		},
		// 8. Plain-text password assignment.
		{
			name: "password_assignment",
			in:   "the user typed password: hunter2!",
			want: "the user typed [REDACTED]",
		},
		// 9. Absolute /home path with user identifier.
		{
			name: "home_path",
			in:   "log written to /home/pedro/secrets.txt yesterday",
			want: "log written to [REDACTED] yesterday",
		},
		// 10. Kaggle working path.
		{
			name: "kaggle_path",
			in:   "checkpoint at /kaggle/working/dcs-gate/auth.json failed",
			want: "checkpoint at [REDACTED] failed",
		},
		// 11. Generic 40+ char opaque token.
		{
			name: "generic_long_token",
			in:   "session id = 1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
			want: "",
		},
		// 12. Negative control: a paragraph with no secrets must pass through unchanged.
		{
			name: "clean_text_preserved",
			in:   "the response shows projected validation followed by a controlled close.",
			want: "the response shows projected validation followed by a controlled close.",
		},
		// 13. Idempotence: an already-redacted chunk must remain stable.
		{
			name: "idempotent",
			in:   "the token was [REDACTED] and the path [REDACTED] was rotated.",
			want: "the token was [REDACTED] and the path [REDACTED] was rotated.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeChunk(c.in)
			if c.want != "" && got != c.want {
				t.Errorf("sanitizeChunk(%q):\n  got  %q\n  want %q", c.in, got, c.want)
				return
			}
			if c.want == "" {
				if !strings.Contains(got, "[REDACTED]") {
					t.Errorf("sanitizeChunk(%q): expected [REDACTED] in result, got %q", c.in, got)
				}
				// Best-effort negative: the raw key prefix shouldn't survive
				// when we expect a redaction. For each prefix family, check
				// it isn't present verbatim in the output.
				forbidden := []string{
					"AKIAIOSFODNN7EXAMPLE",
					"AIzaSyA1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R",
					"ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789",
					"xoxb-FAKE-PLACEHOLDER-NOT-A-REAL-TOKEN",
					"1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
				}
				for _, f := range forbidden {
					if strings.Contains(got, f) {
						t.Errorf("sanitizeChunk(%q): leaked %q in output %q", c.in, f, got)
					}
				}
			}
		})
	}
}

// TestSanitizeChunkOrderIndependence verifies the function works the same
// regardless of which secret appears first when multiple are concatenated.
func TestSanitizeChunkMultiSecret(t *testing.T) {
	in := "first sk-aaaaaaaaaaaaaaaaaaaaaa then password: x then /home/me/foo"
	got := sanitizeChunk(in)
	// Should redact all three: API key, password assignment, home path.
	if strings.Count(got, "[REDACTED]") < 3 {
		t.Errorf("expected at least 3 redactions in %q, got %q", in, got)
	}
	if strings.Contains(got, "sk-aaaaaaaaaaaaaaaaaaaaaa") {
		t.Errorf("openai key leaked through multi-secret input: %q", got)
	}
	if strings.Contains(got, "/home/me/foo") {
		t.Errorf("home path leaked through multi-secret input: %q", got)
	}
}

// TestSanitizeChunkEmpty confirms empty input is safe.
func TestSanitizeChunkEmpty(t *testing.T) {
	if got := sanitizeChunk(""); got != "" {
		t.Errorf("sanitizeChunk(\"\") = %q, want empty string", got)
	}
}
