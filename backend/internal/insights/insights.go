// Package insights serves server-side SQL aggregations over focus_sessions:
// streak, today/week totals, a 98-day heatmap, current-week bars, and a
// by-label donut. Only type='focus' rows count. Behind auth.RequireAuth.
package insights

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"focus/backend/internal/auth"
	"focus/backend/internal/httpx"
)

const dateFmt = "2006-01-02"

// Service holds the pool shared by the insights handler.
type Service struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Register attaches the insights route onto an (auth-protected) router.
func (s *Service) Register(r chi.Router) {
	r.Get("/insights", s.get)
}

func (s *Service) get(w http.ResponseWriter, r *http.Request) {
	loc, err := time.LoadLocation(defaultTZ(r.URL.Query().Get("tz")))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid tz (expected an IANA name like America/New_York)")
		return
	}
	zone := loc.String()
	uid := auth.UserID(r.Context())
	ctx := r.Context()

	heatmap, err := s.heatmap(ctx, uid, zone)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	streak, err := s.streak(ctx, uid, zone)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}
	byLabel, err := s.byLabel(ctx, uid)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database error")
		return
	}

	// Derive today/week totals and the 7 weekday bars from the heatmap (which
	// already ends on today and covers the whole current week).
	byDate := make(map[string]int, len(heatmap))
	for _, b := range heatmap {
		byDate[b.Date] = b.Minutes
	}
	today, _ := time.Parse(dateFmt, heatmap[len(heatmap)-1].Date)
	monday := mondayOf(today)
	weekly := make([]WeekdayBucket, 7)
	weekMin := 0
	for i := range weekly {
		d := monday.AddDate(0, 0, i)
		m := byDate[d.Format(dateFmt)]
		weekly[i] = WeekdayBucket{Weekday: d.Format("Mon"), Minutes: m}
		weekMin += m
	}

	withPct(byLabel)

	httpx.JSON(w, http.StatusOK, Insights{
		Streak:     streak,
		TodayMin:   heatmap[len(heatmap)-1].Minutes,
		WeekMin:    weekMin,
		Heatmap:    heatmap,
		WeeklyBars: weekly,
		ByLabel:    byLabel,
	})
}

// --- queries ---

func (s *Service) heatmap(ctx context.Context, uid int64, zone string) ([]DayBucket, error) {
	rows, err := s.pool.Query(ctx, `
		with bounds as (select ((now() at time zone $2)::date) as today),
		span as (
			select generate_series(
				(select today from bounds) - 97,
				(select today from bounds),
				interval '1 day')::date d),
		mins as (
			select (started_at at time zone $2)::date d, sum(duration_min)::int m
			from focus_sessions
			where user_id = $1 and type = 'focus'
			  and (started_at at time zone $2)::date >= (select today from bounds) - 97
			group by 1)
		select span.d, coalesce(mins.m, 0)
		from span left join mins using (d)
		order by span.d`, uid, zone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DayBucket, 0, 98)
	for rows.Next() {
		var d time.Time
		var m int
		if err := rows.Scan(&d, &m); err != nil {
			return nil, err
		}
		out = append(out, DayBucket{Date: d.Format(dateFmt), Minutes: m})
	}
	return out, rows.Err()
}

func (s *Service) streak(ctx context.Context, uid int64, zone string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		with days as (
			select distinct (started_at at time zone $2)::date d
			from focus_sessions where user_id = $1 and type = 'focus'),
		grp as (select d, d - (row_number() over (order by d))::int as g from days)
		select count(*)::int from grp
		where g = (select g from grp order by d desc limit 1)
		  and (select max(d) from days) >= (now() at time zone $2)::date - 1`,
		uid, zone).Scan(&n)
	return n, err
}

func (s *Service) byLabel(ctx context.Context, uid int64) ([]LabelSlice, error) {
	rows, err := s.pool.Query(ctx, `
		select coalesce(nullif(label, ''), 'Unlabeled') label, sum(duration_min)::int m
		from focus_sessions where user_id = $1 and type = 'focus'
		group by 1 order by m desc`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LabelSlice, 0)
	for rows.Next() {
		var l LabelSlice
		if err := rows.Scan(&l.Label, &l.Minutes); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// --- helpers ---

func defaultTZ(v string) string {
	if v == "" {
		return "UTC"
	}
	return v
}

// mondayOf returns the Monday of the week containing d (a civil date).
func mondayOf(d time.Time) time.Time {
	offset := (int(d.Weekday()) + 6) % 7 // days since Monday (Sun=0 → 6)
	return d.AddDate(0, 0, -offset)
}

// withPct fills each slice's Pct as a share of the total minutes (1 decimal).
func withPct(slices []LabelSlice) {
	total := 0
	for _, l := range slices {
		total += l.Minutes
	}
	if total == 0 {
		return
	}
	for i := range slices {
		slices[i].Pct = math.Round(float64(slices[i].Minutes)/float64(total)*1000) / 10
	}
}
