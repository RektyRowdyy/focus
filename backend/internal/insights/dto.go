package insights

// Response shapes for GET /insights, kept separate from handler/query logic
// (see the codebase structure convention). All ready-to-render for the UI.

// Insights is the full aggregation payload.
type Insights struct {
	Streak     int             `json:"streak"`     // consecutive focus days ending today/yesterday
	TodayMin   int             `json:"todayMin"`   // focus minutes today
	WeekMin    int             `json:"weekMin"`    // focus minutes since Monday
	Heatmap    []DayBucket     `json:"heatmap"`    // last 98 days, oldest→newest, zero-filled
	WeeklyBars []WeekdayBucket `json:"weeklyBars"` // current week Mon→Sun
	ByLabel    []LabelSlice    `json:"byLabel"`    // minutes per label, desc
}

// DayBucket is one day's focus minutes in the heatmap. Date is "2006-01-02".
type DayBucket struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}

// WeekdayBucket is one weekday's focus minutes in the current-week bar chart.
type WeekdayBucket struct {
	Weekday string `json:"weekday"`
	Minutes int    `json:"minutes"`
}

// LabelSlice is one slice of the by-label donut. Pct is a percentage 0–100.
type LabelSlice struct {
	Label   string  `json:"label"`
	Minutes int     `json:"minutes"`
	Pct     float64 `json:"pct"`
}
