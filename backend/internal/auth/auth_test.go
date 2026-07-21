package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// --- always-on unit tests (no DB) ---

func TestNewTokenUniqueAndURLSafe(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		tok, err := newToken()
		if err != nil {
			t.Fatalf("newToken: %v", err)
		}
		if len(tok) < 40 {
			t.Fatalf("token too short: %q", tok)
		}
		if strings.ContainsAny(tok, "+/=") {
			t.Fatalf("token not URL-safe: %q", tok)
		}
		if seen[tok] {
			t.Fatalf("duplicate token: %q", tok)
		}
		seen[tok] = true
	}
}

func TestHashTokenDeterministicAndHidesInput(t *testing.T) {
	h1, h2 := hashToken("abc"), hashToken("abc")
	if h1 != h2 {
		t.Fatal("hashToken not deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(h1))
	}
	if strings.Contains(h1, "abc") {
		t.Fatal("hash leaks the raw token")
	}
	if hashToken("abc") == hashToken("abd") {
		t.Fatal("distinct tokens collided")
	}
}

func TestBcryptRoundtrip(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("hunter2!")); err != nil {
		t.Fatal("correct password rejected")
	}
	if bcrypt.CompareHashAndPassword(hash, []byte("wrong")) == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestDecodeCredsValidation(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantOK   bool
		wantCode int
	}{
		{"ok", `{"email":"A@B.com ","password":"hunter2!"}`, true, http.StatusOK},
		{"bad json", `{`, false, http.StatusBadRequest},
		{"no at sign", `{"email":"nope","password":"hunter2!"}`, false, http.StatusBadRequest},
		{"short password", `{"email":"a@b.com","password":"short"}`, false, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			c, ok := decodeCreds(rec, req)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v (code %d)", ok, tc.wantOK, rec.Code)
			}
			if ok {
				if c.Email != "a@b.com" {
					t.Errorf("email not normalized: %q", c.Email)
				}
			} else if rec.Code != tc.wantCode {
				t.Errorf("code=%d want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

func TestClearCookieAttributes(t *testing.T) {
	rec := httptest.NewRecorder()
	(&Service{}).clearCookie(rec)
	ck := rec.Result().Cookies()[0]
	if ck.Name != cookieName || !ck.HttpOnly || ck.Path != "/" ||
		ck.MaxAge >= 0 || ck.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected clear cookie: %+v", ck)
	}
}

// --- DB-backed flow tests (skip when Postgres is unreachable) ---

func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
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
	srv := httptest.NewServer(New(pool, false).Routes())
	t.Cleanup(func() { srv.Close(); pool.Close() })

	jar, _ := cookiejar.New(nil)
	return srv, &http.Client{Jar: jar}
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

func TestAuthFlow(t *testing.T) {
	srv, client := newTestServer(t)
	email := fmt.Sprintf("t-%d@test.local", time.Now().UnixNano())
	creds := fmt.Sprintf(`{"email":%q,"password":"hunter2!"}`, email)

	if code, body := do(t, client, http.MethodPost, srv.URL+"/signup", creds); code != http.StatusCreated {
		t.Fatalf("signup: %d %s", code, body)
	}
	// Session cookie from signup lets /me through.
	if code, body := do(t, client, http.MethodGet, srv.URL+"/me", ""); code != http.StatusOK ||
		!strings.Contains(body, email) {
		t.Fatalf("me after signup: %d %s", code, body)
	}
	// Duplicate email is rejected.
	if code, _ := do(t, client, http.MethodPost, srv.URL+"/signup", creds); code != http.StatusConflict {
		t.Fatalf("duplicate signup: want 409, got %d", code)
	}
	// Logout invalidates the session server-side.
	if code, _ := do(t, client, http.MethodPost, srv.URL+"/logout", ""); code != http.StatusNoContent {
		t.Fatalf("logout: %d", code)
	}
	if code, _ := do(t, client, http.MethodGet, srv.URL+"/me", ""); code != http.StatusUnauthorized {
		t.Fatalf("me after logout: want 401, got %d", code)
	}
	// Wrong password does not authenticate.
	bad := fmt.Sprintf(`{"email":%q,"password":"wrongpass"}`, email)
	if code, _ := do(t, client, http.MethodPost, srv.URL+"/login", bad); code != http.StatusUnauthorized {
		t.Fatalf("bad login: want 401, got %d", code)
	}
	// Correct login re-issues a working session.
	if code, _ := do(t, client, http.MethodPost, srv.URL+"/login", creds); code != http.StatusOK {
		t.Fatalf("login: %d", code)
	}
	if code, _ := do(t, client, http.MethodGet, srv.URL+"/me", ""); code != http.StatusOK {
		t.Fatalf("me after login: %d", code)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	srv, client := newTestServer(t)
	if code, _ := do(t, client, http.MethodGet, srv.URL+"/me", ""); code != http.StatusUnauthorized {
		t.Fatalf("me without cookie: want 401, got %d", code)
	}
}
