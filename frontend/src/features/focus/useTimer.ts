import { useEffect, useState } from "react";
import { settingsApi } from "../settings/api";
import { DEFAULT_SETTINGS, type Settings } from "../settings/types";
import { sessionsApi } from "../sessions/api";
import {
  complete,
  derive,
  durationMs,
  initState,
  pause,
  reset,
  setType,
  skip,
  start,
  type SessionType,
  type TimerState,
} from "./engine";
import { chime } from "./chime";

// useTimer owns the live timer: a 1s clock tick that only reads Date.now()
// (the remaining time is always derived, never decremented), plus completion
// side-effects (record the focus block, fire the chime, advance the cycle).
export function useTimer() {
  const [settings, setSettings] = useState<Settings>(DEFAULT_SETTINGS);
  const [state, setState] = useState<TimerState>(() => initState(DEFAULT_SETTINGS));
  const [label, setLabel] = useState("");
  const [now, setNow] = useState(() => Date.now());

  // Load the user's real settings; refresh the paused duration without
  // disturbing a running countdown.
  useEffect(() => {
    settingsApi
      .get()
      .then((s) => {
        setSettings(s);
        setState((st) => (st.running ? st : { ...st, remainingMs: durationMs(st.type, s) }));
      })
      .catch(() => {});
  }, []);

  // Clock: tick every second and whenever the tab regains focus/visibility, so
  // remaining is recomputed from timestamps after a backgrounded gap.
  useEffect(() => {
    const sync = () => setNow(Date.now());
    const id = window.setInterval(sync, 1000);
    document.addEventListener("visibilitychange", sync);
    window.addEventListener("focus", sync);
    return () => {
      window.clearInterval(id);
      document.removeEventListener("visibilitychange", sync);
      window.removeEventListener("focus", sync);
    };
  }, []);

  // Completion: when the clock passes the target, record + chime + advance.
  // Setting state re-runs this, so multiple sessions elapsed while hidden catch
  // up one at a time.
  useEffect(() => {
    if (!state.running || state.targetEndTime === null || now < state.targetEndTime) return;
    const { state: next, record } = complete(state, settings, state.targetEndTime);
    if (record) {
      sessionsApi
        .create({
          startedAt: new Date(record.startedAt).toISOString(),
          durationMin: record.durationMin,
          type: "focus",
          label: label.trim() || undefined,
        })
        .catch(() => {});
    }
    if (settings.chime) chime();
    setState(next);
  }, [now, state, settings, label]);

  return {
    type: state.type,
    remainingMs: derive(state, now).remainingMs,
    totalMs: durationMs(state.type, settings),
    running: state.running,
    completedInCycle: state.completedInCycle,
    longInterval: settings.longInterval,
    label,
    setLabel,
    start: () => setState((s) => start(s, Date.now())),
    pause: () => setState((s) => pause(s, Date.now())),
    skip: () => setState((s) => skip(s, settings, Date.now())),
    reset: () => setState((s) => reset(s, settings)),
    setType: (t: SessionType) => setState((s) => setType(s, settings, t)),
  };
}
