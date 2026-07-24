import { describe, it, expect } from "vitest";
import { DEFAULT_SETTINGS, type Settings } from "../settings/types";
import { complete, derive, initState, reset, setType, skip, start } from "./engine";

const S: Settings = { ...DEFAULT_SETTINGS, focusMin: 25, shortMin: 5, longMin: 15, longInterval: 4 };
const MIN = 60_000;

describe("derive — timestamp-based, drift-free", () => {
  it("computes remaining from the clock and survives a backgrounded gap", () => {
    const t0 = 1_000_000;
    const running = start(initState(S), t0); // 25 min session

    // 10 minutes in → exactly 15 min left, no accumulated drift.
    expect(derive(running, t0 + 10 * MIN).remainingMs).toBe(15 * MIN);

    // Jump far past the end (tab was hidden) → clamped to 0 and completed.
    const past = derive(running, t0 + 30 * MIN);
    expect(past.remainingMs).toBe(0);
    expect(past.completed).toBe(true);
  });

  it("holds remaining steady while paused", () => {
    const t0 = 500;
    const paused = initState(S); // paused by default
    expect(derive(paused, t0 + 99 * MIN).remainingMs).toBe(25 * MIN);
    expect(derive(paused, t0 + 99 * MIN).completed).toBe(false);
  });
});

describe("complete — cycle rules", () => {
  it("long break every N focus sessions, then resets the cycle", () => {
    let st = start(initState(S), 0);
    const seq: string[] = [];
    // Walk 4 focus + their breaks.
    for (let i = 0; i < 8; i++) {
      const { state } = complete(st, S, 0);
      seq.push(state.type);
      st = start({ ...state, running: true, targetEndTime: 0, startedAt: 0 }, 0);
    }
    // focus→short, short→focus, ... 4th focus→long, long→focus
    expect(seq).toEqual(["short", "focus", "short", "focus", "short", "focus", "long", "focus"]);
  });

  it("records a focus block but not a break", () => {
    const focus = start(initState(S), 1_000);
    const done = complete(focus, S, 1_000 + 25 * MIN);
    expect(done.record).toEqual({ startedAt: 1_000, durationMin: 25 });

    const brk = start(setType(initState(S), S, "short"), 0);
    expect(complete(brk, S, 5 * MIN).record).toBeNull();
  });
});

describe("setType / reset / skip", () => {
  it("setType switches duration and pauses", () => {
    const st = setType(start(initState(S), 0), S, "long");
    expect(st.type).toBe("long");
    expect(st.running).toBe(false);
    expect(st.remainingMs).toBe(15 * MIN);
  });

  it("reset restores the full current duration, paused", () => {
    const st = reset(start(initState(S), 0), S);
    expect(st.running).toBe(false);
    expect(st.remainingMs).toBe(25 * MIN);
  });

  it("skip advances the cycle without a record", () => {
    const st = skip(start(initState(S), 0), S, 0);
    expect(st.type).toBe("short");
    expect(st.completedInCycle).toBe(1);
  });
});
