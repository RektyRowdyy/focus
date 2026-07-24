// Pure, framework-free timer state machine. The countdown is timestamp-based:
// while running we store targetEndTime and derive `remaining` from Date.now(),
// so it never drifts and survives tab-backgrounding. No React, no DOM here.

import type { Settings } from "../settings/types";

export type SessionType = "focus" | "short" | "long";

export interface TimerState {
  type: SessionType;
  running: boolean;
  targetEndTime: number | null; // epoch ms; set while running
  remainingMs: number; // authoritative while paused
  completedInCycle: number; // focus sessions done toward the long-break interval
  startedAt: number | null; // epoch ms the current focus session began (for recording)
}

// A completed focus block to record via the sessions API.
export interface FocusRecord {
  startedAt: number;
  durationMin: number;
}

export function durationMs(type: SessionType, s: Settings): number {
  const min = type === "focus" ? s.focusMin : type === "short" ? s.shortMin : s.longMin;
  return min * 60_000;
}

export function initState(s: Settings): TimerState {
  return {
    type: "focus",
    running: false,
    targetEndTime: null,
    remainingMs: durationMs("focus", s),
    completedInCycle: 0,
    startedAt: null,
  };
}

// derive is the single source of truth for the clock while running.
export function derive(state: TimerState, now: number): { remainingMs: number; completed: boolean } {
  if (!state.running || state.targetEndTime === null) {
    return { remainingMs: state.remainingMs, completed: false };
  }
  const remainingMs = Math.max(0, state.targetEndTime - now);
  return { remainingMs, completed: remainingMs === 0 };
}

export function start(state: TimerState, now: number): TimerState {
  if (state.running) return state;
  return {
    ...state,
    running: true,
    targetEndTime: now + state.remainingMs,
    startedAt: state.type === "focus" && state.startedAt === null ? now : state.startedAt,
  };
}

export function pause(state: TimerState, now: number): TimerState {
  if (!state.running) return state;
  return {
    ...state,
    running: false,
    remainingMs: derive(state, now).remainingMs,
    targetEndTime: null,
  };
}

// reset returns the current type to its full duration, paused.
export function reset(state: TimerState, s: Settings): TimerState {
  return {
    ...state,
    running: false,
    targetEndTime: null,
    remainingMs: durationMs(state.type, s),
    startedAt: null,
  };
}

// setType switches to a type, resets to its duration, and pauses.
export function setType(state: TimerState, s: Settings, type: SessionType): TimerState {
  return {
    ...state,
    type,
    running: false,
    targetEndTime: null,
    remainingMs: durationMs(type, s),
    startedAt: null,
  };
}

// nextType + cycle bookkeeping when the current session ends (completion or skip).
function advanceCycle(state: TimerState, s: Settings): { type: SessionType; completedInCycle: number } {
  if (state.type === "focus") {
    const completedInCycle = state.completedInCycle + 1;
    const type: SessionType = completedInCycle % s.longInterval === 0 ? "long" : "short";
    return { type, completedInCycle };
  }
  // A break ended → back to focus; a long break starts a fresh cycle.
  const completedInCycle = state.type === "long" ? 0 : state.completedInCycle;
  return { type: "focus", completedInCycle };
}

// buildNext sets up the following session, auto-starting per the settings flags.
function buildNext(
  prevType: SessionType,
  next: { type: SessionType; completedInCycle: number },
  s: Settings,
  now: number,
): TimerState {
  void prevType;
  const running = next.type === "focus" ? s.autoFocus : s.autoBreaks;
  const remainingMs = durationMs(next.type, s);
  return {
    type: next.type,
    running,
    remainingMs,
    completedInCycle: next.completedInCycle,
    targetEndTime: running ? now + remainingMs : null,
    startedAt: running && next.type === "focus" ? now : null,
  };
}

// complete ends the current session naturally. record is set only when a focus
// block finished (so the caller posts it to the sessions API).
export function complete(
  state: TimerState,
  s: Settings,
  now: number,
): { state: TimerState; record: FocusRecord | null } {
  const record: FocusRecord | null =
    state.type === "focus" && state.startedAt !== null
      ? { startedAt: state.startedAt, durationMin: s.focusMin }
      : null;
  const next = advanceCycle(state, s);
  return { state: buildNext(state.type, next, s, now), record };
}

// skip advances through the cycle the same way but records nothing — a skipped
// session wasn't earned.
export function skip(state: TimerState, s: Settings, now: number): TimerState {
  const next = advanceCycle(state, s);
  return buildNext(state.type, next, s, now);
}
