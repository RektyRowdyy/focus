# Focus — Feature & Technical Requirements

A distraction-free focus timer combining Pomodoro sessions, ambient soundscapes, and
progress analytics in one calm, uncluttered interface. Responsive-first for desktop, iPad,
and mobile. Dark mode is the default.

> **Out of scope for this build:** Tasks / Workflow (task list, projects, per-task pomodoro
> tracking, drag-reorder, active-task selection). The prototypes reference a "current task"
> chip — for now treat it as a single free-text label with no list/CRUD behind it, or hide it.

---

## 1. Design system

### Palette
- Near-monochrome base. Dark: background `#0e0f10`, surfaces `rgba(255,255,255,0.03–0.06)`,
  text `rgba(255,255,255,0.4–0.95)` by hierarchy. Light mode mirrors this (off-white base,
  charcoal text) — must be a full theme, not an inversion afterthought.
- **Single accent** `#6aab9d` (sage) — used *only* for active state and progress. Never
  decorative. Hover/active tints derive from it (`rgba(106,171,157,0.14–0.3)`).
- Max 1–2 background tones per screen. No gradients-for-their-own-sake, no shadow stacks —
  depth comes from light (subtle surface lifts), not drop shadows.

### Typography
- One neo-grotesque family (Inter in the prototype). Timer digits: large, weight 200–300,
  tabular numerals, tight negative tracking. Everything else small and low-contrast until
  hovered/focused.

### Motion
- Slow, soft. Fades and eases (~0.3–0.6s), never bounces. Session start/end should feel like
  a breath. Standard easing `ease` / `ease-in-out`; no spring/overshoot.
- Nothing flashes, badges, or nags during an active focus block.

### Shape & depth
- Generous radii (12–26px), thin hairline dividers (`0.5px rgba(255,255,255,0.06)`),
  minimal borders. 44px minimum touch targets on all devices.

---

## 2. Core features

### 2.1 Focus Session (primary)
- Circular progress ring around the countdown (ring is the chosen treatment).
- Session-type toggle: **Focus / Short Break / Long Break**. Switching resets the timer to
  that type's duration and pauses.
- Controls: **start / pause / skip / reset** — icon-only. Revealed on hover (pointer),
  always visible on touch.
- **Countdown** in `M:SS`, tabular, updates once per second.
- **Cycle counter** — dots showing progress through the cycle (e.g. "Session 3 of 4"). Long
  break triggers every _N_ focus sessions (default 4, configurable).
- Subtle current-session label ("Focus" / "Short Break" / "Long Break") above the digits.

### 2.2 Focus Mode (Silence All Distractions)
- One prominent toggle → full-screen zen mode: chrome melts away, only timer + soundscape
  remain.
- Optional slow breathing glow behind the timer (tweakable on/off).
- Do Not Disturb / notification suppression; blocklist for distracting sites/apps.
- **Gentle exit guard** mid-session: confirmation showing elapsed time
  ("You're 12 min into this session…") with Stay / Leave choices. No guard when paused/idle.

### 2.3 Soundscapes
- Five ambiences: **Rain, Forest, Airplane, Café, Clouding**.
- Compact tile row, expandable into a mixer. **Independent volume per sound** so they layer.
- Active sound shows a soft animated indicator. Playback **persists across sessions** and
  across timer state changes.
- **Master mute** always one tap away.

### 2.4 Progress / Insights
- Streak counter; daily and weekly focus totals.
- Heatmap calendar of focused hours (last ~14 weeks).
- Bar chart: focus time by day. Donut/list: time by project or session label.
- Charts stay flat, thin-stroked, largely unlabeled — data as texture, not a report.

### 2.5 Settings
- Customizable durations: focus / short break / long break (minutes).
- Long-break interval (every N sessions).
- Auto-start rules: auto-start breaks, auto-start next focus.
- Completion chime on/off.

---

## 3. Responsive behavior

| Device | Layout |
|---|---|
| **Desktop** | Two-column: timer centered, soundscape dock bottom-left; keyboard shortcuts. |
| **iPad landscape** | Left nav rail (Focus / Insights / Settings + focus-mode), centered timer, soundscape dock bottom-left. Insights & Settings as centered overlays. |
| **iPad portrait** | Single column stacked; secondary panels as a bottom sheet. |
| **Mobile** | Single column, timer fills viewport; soundscapes/insights in bottom tab bar or swipe-up sheets. Thumb-reachable, ≥44px targets. |

- **Keyboard shortcuts** (pointer devices): `Space` start/pause · `R` reset · `S` skip ·
  `Esc` exit zen / close overlay.

---

## 4. Technical requirements

### Timer engine
- Countdown MUST be **timestamp-based**, not `setInterval` decrement — store the target end
  time and derive `remaining` from `Date.now()`. A naive per-second decrement drifts and
  freezes when the tab is backgrounded.
- On session complete: advance type per cycle rules, honor auto-start flags, fire chime if
  enabled. Long break every N focus sessions.
- Timer keeps running (or correctly reflects elapsed time) when the tab is backgrounded;
  recompute `remaining` on `visibilitychange`/focus.

### Audio
- Web Audio API or looping `<audio>` per soundscape; per-source gain nodes for independent
  volume; a master gain for mute. Loop seamlessly.
- Respect browser autoplay policy — start audio only after a user gesture.

### Persistence (localStorage or equivalent)
- Settings (durations, interval, auto-start, chime), theme choice, active soundscapes +
  per-sound volumes + master mute, and session-history data feeding Insights.
- Restore all of the above on load. Never clobber existing keys.

### Notifications / DND (Focus Mode)
- Notification suppression + site/app blocklist. Web build: Notifications API + a
  best-effort blocklist UI; native/desktop wrapper if targeting real DND.

### Theming
- Full light + dark via CSS custom properties (or equivalent tokens). Dark is default;
  respect and allow overriding `prefers-color-scheme`.

### Data model (Insights)
- Persist completed focus sessions with `{ startedAt, durationMin, type, label? }`.
- Derive streak, daily/weekly totals, heatmap buckets, and by-label donut from that log.

### Non-functional
- Fast and gestural on every device. Idle CPU near-zero (no busy repaints while paused).
- Accessible: focus states, ARIA on icon-only controls, reduced-motion support
  (`prefers-reduced-motion` disables the breathing glow and softens transitions).

---

## 5. Guiding principles
- The app should feel calmer than the work it's helping with.
- Every element must justify its pixels — if it doesn't serve the current session, hide it.
- Nothing flashes, badges, or nags during an active focus block.
- Fast, gestural, and quiet on every device.

---

## 6. Suggested build order
1. Timer engine (timestamp-based) + Focus Session UI + session-type toggle.
2. Settings + persistence; wire durations/auto-start/interval into the engine.
3. Soundscapes (audio graph, per-source volume, master mute, persistence).
4. Focus Mode (zen full-screen, breathing glow, exit guard, DND/blocklist).
5. Insights (session log → streak, totals, heatmap, bar, donut).
6. Responsive passes: desktop → iPad landscape → iPad portrait → mobile.
7. Theming (light/dark), accessibility, reduced-motion.
