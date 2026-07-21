// Package auth provides email+password signup/login/logout backed by httpOnly
// session cookies, plus a RequireAuth middleware guarding protected routes.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"focus/backend/internal/httpx"
)

const (
	cookieName = "focus_session"
	sessionTTL = 30 * 24 * time.Hour
	minPwLen   = 8
)

// Service holds the dependencies shared by the auth handlers and middleware.
type Service struct {
	pool         *pgxpool.Pool
	cookieSecure bool
}

// New builds an auth service over the given pool. cookieSecure sets the cookie
// Secure flag (leave false for local http dev).
func New(pool *pgxpool.Pool, cookieSecure bool) *Service {
	return &Service{pool: pool, cookieSecure: cookieSecure}
}

// Routes returns the /auth sub-router.
func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/signup", s.signup)
	r.Post("/login", s.login)
	r.Post("/logout", s.logout)
	r.With(s.RequireAuth).Get("/me", s.me)
	return r
}

// --- handlers ---

func (s *Service) signup(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCreds(w, r)
	if !ok {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var id int64
	err = tx.QueryRow(ctx,
		`insert into users (email, password_hash) values ($1, $2) returning id`,
		c.Email, string(hash),
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			httpx.Error(w, http.StatusConflict, "email already registered")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "could not create user")
		return
	}
	// Seed default settings; column defaults fill in the values.
	if _, err = tx.Exec(ctx, `insert into user_settings (user_id) values ($1)`, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create settings")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not create user")
		return
	}

	if err := s.issueSession(ctx, w, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not start session")
		return
	}
	httpx.JSON(w, http.StatusCreated, userView{ID: id, Email: c.Email})
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	c, ok := decodeCreds(w, r)
	if !ok {
		return
	}
	var (
		id   int64
		hash string
	)
	err := s.pool.QueryRow(r.Context(),
		`select id, password_hash from users where email = $1`, c.Email,
	).Scan(&id, &hash)
	// Same 401 for unknown email and wrong password — no user enumeration.
	if errors.Is(err, pgx.ErrNoRows) ||
		(err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(c.Password)) != nil) {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := s.issueSession(r.Context(), w, id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not start session")
		return
	}
	httpx.JSON(w, http.StatusOK, userView{ID: id, Email: c.Email})
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	if ck, err := r.Cookie(cookieName); err == nil {
		_, _ = s.pool.Exec(r.Context(),
			`delete from sessions_auth where token = $1`, hashToken(ck.Value))
	}
	s.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	id := UserID(r.Context())
	var email string
	if err := s.pool.QueryRow(r.Context(),
		`select email from users where id = $1`, id).Scan(&email); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	httpx.JSON(w, http.StatusOK, userView{ID: id, Email: email})
}

// --- middleware + context ---

type ctxKey int

const userIDKey ctxKey = 0

// RequireAuth rejects requests without a valid, unexpired session cookie and
// stashes the authenticated user id in the request context.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie(cookieName)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var id int64
		err = s.pool.QueryRow(r.Context(),
			`select user_id from sessions_auth where token = $1 and expires_at > now()`,
			hashToken(ck.Value),
		).Scan(&id)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserID returns the authenticated user id from a context passed through
// RequireAuth. Zero if unset (never for RequireAuth-guarded handlers).
func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

// --- helpers ---

func decodeCreds(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return c, false
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if !strings.Contains(c.Email, "@") {
		httpx.Error(w, http.StatusBadRequest, "a valid email is required")
		return c, false
	}
	if len(c.Password) < minPwLen {
		httpx.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return c, false
	}
	return c, true
}

// issueSession creates a session row (storing only the token hash) and sets the cookie.
func (s *Service) issueSession(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	expires := time.Now().Add(sessionTTL)
	if _, err := s.pool.Exec(ctx,
		`insert into sessions_auth (token, user_id, expires_at) values ($1, $2, $3)`,
		hashToken(token), userID, expires,
	); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Service) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// newToken returns a 256-bit URL-safe random session token.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken is what we persist — a DB read then can't replay a live session.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
