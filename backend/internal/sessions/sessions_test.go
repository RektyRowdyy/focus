package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"focus/backend/internal/auth"
)

// --- no-DB validation tests ---

func ptr[T any](v T) *T { return &v }

func TestValidateSession(t *testing.T) {
	now := time.Now()
	long := strings.Repeat("x", 201)
	tests := []struct {
		name    string
		in      sessionInput
		wantErr bool
	}{
		{"ok focus", sessionInput{StartedAt: now, DurationMin: 25, Type: "focus"}, false},
		{"ok short", sessionInput{StartedAt: now, DurationMin: 5, Type: "short"}, false},
		{"zero startedAt", sessionInput{DurationMin: 25, Type: "focus"}, true},
		{"duration zero", sessionInput{StartedAt: now, DurationMin: 0, Type: "focus"}, true},
		{"duration over", sessionInput{StartedAt: now, DurationMin: 1441, Type: "focus"}, true},
		{"bad type", sessionInput{StartedAt: now, DurationMin: 25, Type: "nope"}, true},
		{"empty type", sessionInput{StartedAt: now, DurationMin: 25}, true},
		{"long label", sessionInput{StartedAt: now, DurationMin: 25, Type: "focus", Label: ptr(long)}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateSession(tc.in) != ""; got != tc.wantErr {
				t.Fatalf("validateSession=%q wantErr=%v", validateSession(tc.in), tc.wantErr)
			}
		})
	}
}

// --- DB-backed tests (skip when Postgres is unreachable) ---

func newAuthedClient(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://focus:focus@localhost:5432/focus?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("database unreachable: %v", err)
	}

	authSvc := auth.New(pool, false)
	r := chi.NewRouter()
	r.Mount("/auth", authSvc.Routes())
	r.Group(func(pr chi.Router) {
		pr.Use(authSvc.RequireAuth)
		New(pool).Register(pr)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(func() { srv.Close(); pool.Close() })

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	email := fmt.Sprintf("ss-%d@test.local", time.Now().UnixNano())
	body := fmt.Sprintf(`{"email":%q,"password":"hunter2!"}`, email)
	if code, b := do(t, client, http.MethodPost, srv.URL+"/auth/signup", body); code != http.StatusCreated {
		t.Fatalf("signup: %d %s", code, b)
	}
	return srv, client
}

func do(t *testing.T, c *http.Client, method, url, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestSessionsFlow(t *testing.T) {
	srv, client := newAuthedClient(t)

	// Create a focus block; the row comes back with an id and echoes the input.
	code, body := do(t, client, http.MethodPost, srv.URL+"/sessions",
		`{"startedAt":"2026-07-22T09:00:00Z","durationMin":25,"type":"focus","label":"deep work"}`)
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, body)
	}
	var s Session
	if err := json.Unmarshal([]byte(body), &s); err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 || s.DurationMin != 25 || s.Type != "focus" || s.Label == nil || *s.Label != "deep work" {
		t.Fatalf("unexpected created session: %+v", s)
	}

	// A later session, then GET must be newest-first.
	do(t, client, http.MethodPost, srv.URL+"/sessions",
		`{"startedAt":"2026-07-22T10:00:00Z","durationMin":25,"type":"focus"}`)
	code, body = do(t, client, http.MethodGet, srv.URL+"/sessions?limit=10", "")
	if code != http.StatusOK {
		t.Fatalf("list: %d %s", code, body)
	}
	var list []Session
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 || !list[0].StartedAt.After(list[1].StartedAt) {
		t.Fatalf("not newest-first: %+v", list)
	}

	// from filter narrows the range.
	code, body = do(t, client, http.MethodGet, srv.URL+"/sessions?from=2026-07-22T09:30:00Z", "")
	if code != http.StatusOK {
		t.Fatalf("list filtered: %d %s", code, body)
	}
	_ = json.Unmarshal([]byte(body), &list)
	for _, se := range list {
		if se.StartedAt.Before(time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)) {
			t.Fatalf("from filter leaked older row: %+v", se)
		}
	}

	// Invalid type rejected.
	if code, _ = do(t, client, http.MethodPost, srv.URL+"/sessions",
		`{"startedAt":"2026-07-22T09:00:00Z","durationMin":25,"type":"nope"}`); code != http.StatusBadRequest {
		t.Fatalf("bad type: want 400, got %d", code)
	}
}

func TestRequiresAuth(t *testing.T) {
	srv, _ := newAuthedClient(t)
	if code, _ := do(t, &http.Client{}, http.MethodGet, srv.URL+"/sessions", ""); code != http.StatusUnauthorized {
		t.Fatalf("no cookie: want 401, got %d", code)
	}
}
