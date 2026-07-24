// Minimal completion chime — a short, soft sine tone via Web Audio. The full
// soundscape graph is T-10; this is just the session-end cue.

let ctx: AudioContext | null = null;

// Lazily create the context (must follow a user gesture per autoplay policy;
// starting the timer counts as one).
function audio(): AudioContext {
  if (!ctx) ctx = new AudioContext();
  return ctx;
}

export function chime(): void {
  try {
    const ac = audio();
    const now = ac.currentTime;
    const osc = ac.createOscillator();
    const gain = ac.createGain();

    osc.type = "sine";
    osc.frequency.setValueAtTime(660, now); // a calm, mid tone
    // Gentle attack + decay so it fades like a breath, never a harsh beep.
    gain.gain.setValueAtTime(0, now);
    gain.gain.linearRampToValueAtTime(0.15, now + 0.04);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.9);

    osc.connect(gain).connect(ac.destination);
    osc.start(now);
    osc.stop(now + 0.95);
  } catch {
    // Audio unavailable (e.g. tests / no gesture) — a missing chime is harmless.
  }
}
