package main

import "testing"

func TestSanitizeChunk(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "openai key",
			in:   "found key sk-abc123def456ghi789jkl012 in env",
			want: "found key [REDACTED] in env",
		},
		{
			name: "aws access key",
			in:   "AKIAIOSFODNN7EXAMPLE is the key",
			want: "[REDACTED] is the key",
		},
		{
			name: "google api key",
			in:   "key AIzaSyA1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q",
			want: "key [REDACTED]",
		},
		{
			name: "github pat",
			in:   "token ghp_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKL",
			want: "token [REDACTED]",
		},
		{
			name: "home path",
			in:   "wrote to /home/pedro/notebook.ipynb successfully",
			want: "wrote to [REDACTED] successfully",
		},
		{
			name: "kaggle path",
			in:   "loaded /kaggle/working/data.json fine",
			want: "loaded [REDACTED] fine",
		},
		{
			name: "password assignment",
			in:   "config: password = hunter2 fail",
			want: "config: [REDACTED] fail",
		},
		{
			name: "secret assignment",
			in:   "api_key: ABC123XYZ",
			want: "[REDACTED]",
		},
		{
			name: "plain innocent text",
			in:   "the response is moderate predictability formulaic false",
			want: "the response is moderate predictability formulaic false",
		},
		{
			name: "multiple patterns in single chunk",
			in:   "found sk-aaaaaaaaaaaaaaaaaaaaaaaa and AKIAIOSFODNN7EXAMPLE in /home/x/y",
			want: "found [REDACTED] and [REDACTED] in [REDACTED]",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizeChunk(c.in)
			if got != c.want {
				t.Errorf("sanitizeChunk(%q):\n  got  %q\n  want %q", c.in, got, c.want)
			}
		})
	}
}
