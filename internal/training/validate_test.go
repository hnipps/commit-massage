package training

import (
	"strings"
	"testing"
)

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantOK  bool
		wantRsn string
	}{
		{
			name:   "simple valid",
			msg:    "feat: add login endpoint",
			wantOK: true,
		},
		{
			name:   "scoped valid",
			msg:    "fix(auth): handle expired tokens",
			wantOK: true,
		},
		{
			name:   "breaking change",
			msg:    "refactor!: drop legacy api",
			wantOK: true,
		},
		{
			name:   "scoped breaking change",
			msg:    "feat(api)!: change response format",
			wantOK: true,
		},
		{
			name:   "with body",
			msg:    "feat: add caching\n\nThis improves response times by caching\nfrequently accessed data.",
			wantOK: true,
		},
		{
			name:    "uppercase description",
			msg:     "feat: Add login endpoint",
			wantRsn: "subject-format",
		},
		{
			name:    "wrong type",
			msg:     "feature: add endpoint",
			wantRsn: "subject-format",
		},
		{
			name:    "missing space after colon",
			msg:     "feat:add endpoint",
			wantRsn: "subject-format",
		},
		{
			name:    "trailing period",
			msg:     "feat: add endpoint.",
			wantRsn: "subject-trailing-period",
		},
		{
			name:    "subject too long",
			msg:     "feat: add " + strings.Repeat("x", 63),
			wantRsn: "subject-too-long",
		},
		{
			name:    "body missing blank separator",
			msg:     "feat: add caching\nsome body text",
			wantRsn: "body-no-blank-line",
		},
		{
			name:    "body line too long",
			msg:     "feat: add caching\n\n" + string(make([]byte, 73)),
			wantRsn: "body-line-too-long",
		},
		{
			name:    "plain message",
			msg:     "Update the readme file",
			wantRsn: "subject-format",
		},
		{
			name:    "uppercase scope",
			msg:     "feat(Auth): add endpoint",
			wantRsn: "subject-format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := ValidateMessage(tt.msg)
			if tt.wantOK {
				if reason != "" {
					t.Errorf("expected valid, got reason %q", reason)
				}
			} else {
				if reason == "" {
					t.Error("expected invalid, got valid")
				}
				if reason != tt.wantRsn {
					t.Errorf("reason: got %q, want %q", reason, tt.wantRsn)
				}
			}
		})
	}
}
