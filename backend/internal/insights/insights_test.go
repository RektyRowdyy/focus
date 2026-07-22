package insights

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
	"focus/backend/internal/sessions"
)

// --- no-DB unit test ---

func TestMondayOf(t *testing.T) {
	for i := range 21 {
		d := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		m := mondayOf(d)
		if m.Weekday() != time.Monday {
			t.Fatalf("mondayOf(%s).Weekday()=%s", d.Format(dateFmt), m.Weekday())
		}
		if diff := d.Sub(m).Hours() / 24; diff < 0 || diff > 6 {
			t.Fatalf("mondayOf(%s) off by %.0f days", d.Format(dateFmt), diff)
		}
	}
}

// --- DB-backed aggregation test ---

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
		sessions.New(pool).Register(pr) // to seed the fixture via the real API
		New(pool).Register(pr)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(func() { srv.Close(); pool.Close() })

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	email := fmt.Sprintf("in-%d@test.local", time.Now().UnixNano())
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

func TestInsightsAggregation(t *testing.T) {
	srv, client := newAuthedClient(t)
	now := time.Now().UTC()
	at := func(offset int) string { return now.AddDate(0, 0, offset).Format(dateFmt) + "T12:00:00Z" }
	post := func(started string, dur int, typ, label string) {
		body := fmt.Sprintf(`{"startedAt":%q,"durationMin":%d,"type":%q,"label":%q}`, started, dur, typ, label)
		if code, b := do(t, client, http.MethodPost, srv.URL+"/sessions", body); code != http.StatusCreated {
			t.Fatalf("seed session: %d %s", code, b)
		}
	}
	// Consecutive focus days today, -1, -2 (streak 3); -4 is an older island.
	post(at(0), 25, "focus", "deep work")
	post(at(0), 25, "focus", "email") // today = 50 min across 2 labels
	post(at(-1), 25, "focus", "deep work")
	post(at(-2), 25, "focus", "deep work")
	post(at(-4), 25, "focus", "deep work")
	post(at(0), 5, "short", "") // a break — must be excluded everywhere

	code, body := do(t, client, http.MethodGet, srv.URL+"/insights?tz=UTC", "")
	if code != http.StatusOK {
		t.Fatalf("insights: %d %s", code, body)
	}
	var ins Insights
	if err := json.Unmarshal([]byte(body), &ins); err != nil {
		t.Fatal(err)
	}

	if ins.Streak != 3 {
		t.Errorf("streak = %d, want 3", ins.Streak)
	}
	if ins.TodayMin != 50 {
		t.Errorf("todayMin = %d, want 50", ins.TodayMin)
	}
	if len(ins.Heatmap) != 98 {
		t.Errorf("heatmap len = %d, want 98", len(ins.Heatmap))
	}
	if last := ins.Heatmap[len(ins.Heatmap)-1]; last.Date != now.Format(dateFmt) || last.Minutes != 50 {
		t.Errorf("last heatmap bucket = %+v, want today/50", last)
	}
	if len(ins.WeeklyBars) != 7 || ins.WeeklyBars[0].Weekday != "Mon" {
		t.Errorf("weeklyBars = %+v, want 7 Mon-first", ins.WeeklyBars)
	}
	weeklySum := 0
	for _, b := range ins.WeeklyBars {
		weeklySum += b.Minutes
	}
	if weeklySum != ins.WeekMin {
		t.Errorf("weeklyBars sum %d != weekMin %d", weeklySum, ins.WeekMin)
	}
	// Break excluded, no Unlabeled: exactly two labelled slices, pct sums ~100.
	if len(ins.ByLabel) != 2 {
		t.Fatalf("byLabel = %+v, want 2 slices (break excluded)", ins.ByLabel)
	}
	if ins.ByLabel[0].Label != "deep work" || ins.ByLabel[0].Minutes != 100 || ins.ByLabel[0].Pct != 80 {
		t.Errorf("byLabel[0] = %+v, want deep work/100/80", ins.ByLabel[0])
	}
	if ins.ByLabel[1].Minutes != 25 || ins.ByLabel[1].Pct != 20 {
		t.Errorf("byLabel[1] = %+v, want 25/20", ins.ByLabel[1])
	}
}

func TestInsightsBadTZAndAuth(t *testing.T) {
	srv, client := newAuthedClient(t)
	if code, _ := do(t, client, http.MethodGet, srv.URL+"/insights?tz=Mars/Olympus", ""); code != http.StatusBadRequest {
		t.Errorf("bad tz: want 400, got %d", code)
	}
	if code, _ := do(t, &http.Client{}, http.MethodGet, srv.URL+"/insights", ""); code != http.StatusUnauthorized {
		t.Errorf("no cookie: want 401, got %d", code)
	}
}
