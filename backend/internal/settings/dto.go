package settings

import "slices"

// Request/response shapes for the settings + soundscape endpoints, kept
// separate from handler/service logic (see the codebase structure convention).

// Settings is the full per-user settings response (camelCase for the TS client).
type Settings struct {
	FocusMin     int    `json:"focusMin"`
	ShortMin     int    `json:"shortMin"`
	LongMin      int    `json:"longMin"`
	LongInterval int    `json:"longInterval"`
	AutoBreaks   bool   `json:"autoBreaks"`
	AutoFocus    bool   `json:"autoFocus"`
	Chime        bool   `json:"chime"`
	Theme        string `json:"theme"`
	MasterMute   bool   `json:"masterMute"`
}

// settingsUpdate is the PUT /settings body. Pointer fields = partial update
// (nil leaves the stored value unchanged).
type settingsUpdate struct {
	FocusMin     *int    `json:"focusMin"`
	ShortMin     *int    `json:"shortMin"`
	LongMin      *int    `json:"longMin"`
	LongInterval *int    `json:"longInterval"`
	AutoBreaks   *bool   `json:"autoBreaks"`
	AutoFocus    *bool   `json:"autoFocus"`
	Chime        *bool   `json:"chime"`
	Theme        *string `json:"theme"`
	MasterMute   *bool   `json:"masterMute"`
}

// SoundscapePref is one sound's state in the GET/PUT /soundscapes list.
type SoundscapePref struct {
	Key     string  `json:"key"`
	Enabled bool    `json:"enabled"`
	Volume  float64 `json:"volume"`
}

// soundscapeUpdate is one entry in the PUT /soundscapes body. Enabled/Volume are
// pointers so a caller can change just one (nil leaves it unchanged).
type soundscapeUpdate struct {
	Key     string   `json:"key"`
	Enabled *bool    `json:"enabled"`
	Volume  *float64 `json:"volume"`
}

// soundKeys is the canonical, ordered set of soundscapes. GET fills any missing
// rows from this list so a new user needs no seeded soundscape rows.
var soundKeys = []string{"rain", "forest", "airplane", "cafe", "clouding"}

func knownSound(key string) bool {
	return slices.Contains(soundKeys, key)
}
