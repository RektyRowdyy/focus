package sessions

import (
	"slices"
	"time"
)

// Request/response shapes for the sessions endpoints, kept separate from
// handler/service logic (see the codebase structure convention).

// Session is a completed focus block as returned to the client.
type Session struct {
	ID          int64     `json:"id"`
	StartedAt   time.Time `json:"startedAt"`
	DurationMin int       `json:"durationMin"`
	Type        string    `json:"type"`
	Label       *string   `json:"label,omitempty"`
}

// sessionInput is the POST /sessions body. time.Time parses RFC3339 from JSON.
type sessionInput struct {
	StartedAt   time.Time `json:"startedAt"`
	DurationMin int       `json:"durationMin"`
	Type        string    `json:"type"`
	Label       *string   `json:"label"`
}

// validTypes matches the focus_sessions.type CHECK constraint. Only "focus"
// blocks count toward insights (T-06 filters on it); breaks are stored too.
var validTypes = []string{"focus", "short", "long"}

const maxLabelLen = 200

// validateSession returns an error message, or "" when the input is valid.
func validateSession(in sessionInput) string {
	if in.StartedAt.IsZero() {
		return "startedAt is required (RFC3339)"
	}
	if in.DurationMin < 1 || in.DurationMin > 1440 {
		return "durationMin must be between 1 and 1440"
	}
	if !slices.Contains(validTypes, in.Type) {
		return "type must be one of focus, short, long"
	}
	if in.Label != nil && len(*in.Label) > maxLabelLen {
		return "label must be at most 200 characters"
	}
	return ""
}
