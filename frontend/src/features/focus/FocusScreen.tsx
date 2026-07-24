import { useEffect } from "react";
import type { SessionType } from "./engine";
import { useTimer } from "./useTimer";
import "./focus.css";

const R = 118;
const C = 2 * Math.PI * R;

const TYPES: { id: SessionType; label: string }[] = [
  { id: "focus", label: "Focus" },
  { id: "short", label: "Short Break" },
  { id: "long", label: "Long Break" },
];

const TYPE_LABEL: Record<SessionType, string> = {
  focus: "Focus",
  short: "Short Break",
  long: "Long Break",
};

function mmss(remainingMs: number): string {
  const total = Math.ceil(remainingMs / 1000);
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export function FocusScreen() {
  const t = useTimer();
  const frac = t.totalMs > 0 ? t.remainingMs / t.totalMs : 0;
  const dashOffset = C * (1 - frac);

  // Keyboard shortcuts (ignored while typing the task label).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return;
      if (e.code === "Space") {
        e.preventDefault();
        t.running ? t.pause() : t.start();
      } else if (e.key === "r" || e.key === "R") {
        t.reset();
      } else if (e.key === "s" || e.key === "S") {
        t.skip();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [t]);

  return (
    <div className="focus">
      <div className="focus__types" role="group" aria-label="Session type">
        {TYPES.map((ty) => (
          <button
            key={ty.id}
            aria-pressed={t.type === ty.id}
            onClick={() => t.setType(ty.id)}
          >
            {ty.label}
          </button>
        ))}
      </div>

      <div className="focus__ring">
        <svg viewBox="0 0 260 260" width="260" height="260" aria-hidden="true">
          <circle className="focus__track" cx="130" cy="130" r={R} />
          <circle
            className="focus__prog"
            cx="130"
            cy="130"
            r={R}
            strokeDasharray={C}
            strokeDashoffset={dashOffset}
          />
        </svg>
        <div className="focus__center">
          <span className="focus__label-type">{TYPE_LABEL[t.type]}</span>
          <span className="focus__digits" role="timer" aria-live="off">
            {mmss(t.remainingMs)}
          </span>
        </div>
      </div>

      <div className="focus__dots" aria-label={`Session ${Math.min(t.completedInCycle + 1, t.longInterval)} of ${t.longInterval}`}>
        {Array.from({ length: t.longInterval }, (_, i) => (
          <span key={i} className={`focus__dot ${i < t.completedInCycle ? "focus__dot--on" : ""}`} />
        ))}
      </div>

      <div className="focus__controls">
        <button
          className="focus__secondary"
          onClick={t.reset}
          aria-label="Reset"
          title="Reset (R)"
        >
          <ResetIcon />
        </button>
        <button
          className="focus__primary"
          onClick={() => (t.running ? t.pause() : t.start())}
          aria-label={t.running ? "Pause" : "Start"}
          title={t.running ? "Pause (Space)" : "Start (Space)"}
        >
          {t.running ? <PauseIcon /> : <PlayIcon />}
        </button>
        <button
          className="focus__secondary"
          onClick={t.skip}
          aria-label="Skip"
          title="Skip (S)"
        >
          <SkipIcon />
        </button>
      </div>

      <input
        className="focus__task"
        value={t.label}
        onChange={(e) => t.setLabel(e.target.value)}
        placeholder="What are you focusing on?"
        aria-label="Current task"
        maxLength={200}
      />
    </div>
  );
}

// --- icon-only controls (inline SVG, no icon lib) ---

const svg = { width: 20, height: 20, viewBox: "0 0 24 24", fill: "currentColor" } as const;

function PlayIcon() {
  return (
    <svg {...svg}>
      <path d="M8 5v14l11-7z" />
    </svg>
  );
}
function PauseIcon() {
  return (
    <svg {...svg}>
      <path d="M6 5h4v14H6zM14 5h4v14h-4z" />
    </svg>
  );
}
function SkipIcon() {
  return (
    <svg {...svg}>
      <path d="M6 5v14l9-7zM16 5h3v14h-3z" />
    </svg>
  );
}
function ResetIcon() {
  return (
    <svg {...svg} fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 12a9 9 0 1 0 3-6.7L3 8" />
      <path d="M3 3v5h5" />
    </svg>
  );
}
