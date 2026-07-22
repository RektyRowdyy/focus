// Package sessions records completed focus blocks and lists them for the authed
// user. Routes are user-scoped via auth.UserID, mounted behind auth.RequireAuth.
package sessions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"focus/backend/internal/auth"
	"focus/backend/internal/httpx"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

var errBadTime = errors.New("from/to must be RFC3339 timestamps")

// Service holds the pool shared by the sessions handlers.
type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Register attaches the sessions routes onto an (auth-protected) router.
func (s *Service) Register(r chi.Router) {
	r.Post("/sessions", s.create)
	r.Get("/sessions", s.list)
}

func (s *Service) create(w http.ResponseWriter, r *http.Request) {
	var in sessionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateSession(in); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	// Trim label; store NULL when blank.
	var label *string
	if in.Label != nil {
		if t := strings.TrimSpace(*in.Label); t != "" {
			label = &t
		}
	}

	var out Session
	err := s.pool.QueryRow(r.Context(),
		`insert into focus_sessions (user_id, started_at, duration_min, type, label)
		 values ($1, $2, $3, $4, $5)
		 returning id, started_at, duration_min, type, label`,
		auth.UserID(r.Context()), in.StartedAt, in.DurationMin, in.Type, label,
	).Scan(&out.ID, &out.StartedAt, &out.DurationMin, &out.Type, &out.Label)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not save session")
		return
	}
	httpx.JSON(w, http.StatusCreated, out)
}

func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseRange(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"))

	rows, err := s.pool.Query(r.Context(),
		`select id, started_at, duration_min, type, label from focus_sessions
		 where user_id = $1
		   and ($2::timestamptz is null or started_at >= $2)
		   and ($3::timestamptz is null or started_at <  $3)
		 order by started_at desc
		 limit $4`,
		auth.UserID(r.Context()), from, to, limit,
	)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	out := make([]Session, 0)
	for rows.Next() {
		var se Session
		if err := rows.Scan(&se.ID, &se.StartedAt, &se.DurationMin, &se.Type, &se.Label); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "database error")
			return
		}
		out = append(out, se)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// parseRange reads optional from/to RFC3339 query params (nil = no bound).
func parseRange(r *http.Request) (from, to *time.Time, err error) {
	q := r.URL.Query()
	if from, err = parseTime(q.Get("from")); err != nil {
		return nil, nil, err
	}
	if to, err = parseTime(q.Get("to")); err != nil {
		return nil, nil, err
	}
	return from, to, nil
}

func parseTime(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, errBadTime
	}
	return &t, nil
}

func parseLimit(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
