package settings

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

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name    string
		in      settingsUpdate
		wantErr bool
	}{
		{"empty ok", settingsUpdate{}, false},
		{"focus min ok", settingsUpdate{FocusMin: ptr(1)}, false},
		{"focus max ok", settingsUpdate{FocusMin: ptr(90)}, false},
		{"focus zero", settingsUpdate{FocusMin: ptr(0)}, true},
		{"focus over", settingsUpdate{FocusMin: ptr(91)}, true},
		{"short over", settingsUpdate{ShortMin: ptr(91)}, true},
		{"interval zero", settingsUpdate{LongInterval: ptr(0)}, true},
		{"interval over", settingsUpdate{LongInterval: ptr(13)}, true},
		{"theme dark", settingsUpdate{Theme: ptr("dark")}, false},
		{"theme light", settingsUpdate{Theme: ptr("light")}, false},
		{"theme junk", settingsUpdate{Theme: ptr("neon")}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateSettings(tc.in) != ""; got != tc.wantErr {
				t.Fatalf("validateSettings=%q wantErr=%v", validateSettings(tc.in), tc.wantErr)
			}
		})
	}
}

func TestKnownSound(t *testing.T) {
	for _, k := range soundKeys {
		if !knownSound(k) {
			t.Errorf("%q should be known", k)
		}
	}
	if knownSound("bagpipes") {
		t.Error("bagpipes should not be known")
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
		pr.Mount("/", New(pool).Routes())
	})
	srv := httptest.NewServer(r)
	t.Cleanup(func() { srv.Close(); pool.Close() })

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	email := fmt.Sprintf("s-%d@test.local", time.Now().UnixNano())
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

func TestSettingsFlow(t *testing.T) {
	srv, client := newAuthedClient(t)

	// GET returns seeded defaults.
	code, body := do(t, client, http.MethodGet, srv.URL+"/settings", "")
	if code != http.StatusOK {
		t.Fatalf("get settings: %d %s", code, body)
	}
	var set Settings
	if err := json.Unmarshal([]byte(body), &set); err != nil {
		t.Fatal(err)
	}
	if set.FocusMin != 25 || set.ShortMin != 5 || set.LongMin != 15 || set.LongInterval != 4 || set.Theme != "dark" {
		t.Fatalf("unexpected defaults: %+v", set)
	}

	// PUT partial: only focusMin + theme change, shortMin stays.
	code, body = do(t, client, http.MethodPut, srv.URL+"/settings", `{"focusMin":50,"theme":"light"}`)
	if code != http.StatusOK {
		t.Fatalf("put settings: %d %s", code, body)
	}
	_ = json.Unmarshal([]byte(body), &set)
	if set.FocusMin != 50 || set.Theme != "light" || set.ShortMin != 5 {
		t.Fatalf("partial update wrong: %+v", set)
	}

	// Out-of-range rejected.
	if code, _ = do(t, client, http.MethodPut, srv.URL+"/settings", `{"focusMin":0}`); code != http.StatusBadRequest {
		t.Fatalf("range validation: want 400, got %d", code)
	}
}

func TestSoundscapesFlow(t *testing.T) {
	srv, client := newAuthedClient(t)

	// GET returns all five canonical sounds, default off.
	code, body := do(t, client, http.MethodGet, srv.URL+"/soundscapes", "")
	if code != http.StatusOK {
		t.Fatalf("get soundscapes: %d %s", code, body)
	}
	var prefs []SoundscapePref
	if err := json.Unmarshal([]byte(body), &prefs); err != nil {
		t.Fatal(err)
	}
	if len(prefs) != len(soundKeys) || prefs[0].Key != "rain" || prefs[0].Enabled || prefs[0].Volume != 0.5 {
		t.Fatalf("unexpected soundscape defaults: %+v", prefs)
	}

	// PUT persists and merges.
	code, body = do(t, client, http.MethodPut, srv.URL+"/soundscapes", `[{"key":"rain","enabled":true,"volume":0.6}]`)
	if code != http.StatusOK {
		t.Fatalf("put soundscapes: %d %s", code, body)
	}
	_ = json.Unmarshal([]byte(body), &prefs)
	if !prefs[0].Enabled || prefs[0].Volume != 0.6 {
		t.Fatalf("rain not updated: %+v", prefs[0])
	}

	// Unknown key rejected.
	if code, _ = do(t, client, http.MethodPut, srv.URL+"/soundscapes", `[{"key":"bagpipes"}]`); code != http.StatusBadRequest {
		t.Fatalf("unknown sound: want 400, got %d", code)
	}
}

func TestRequiresAuth(t *testing.T) {
	srv, _ := newAuthedClient(t)
	// A fresh client with no session cookie is rejected.
	if code, _ := do(t, &http.Client{}, http.MethodGet, srv.URL+"/settings", ""); code != http.StatusUnauthorized {
		t.Fatalf("no cookie: want 401, got %d", code)
	}
}
