import type { CSSProperties } from "react";

// Placeholder dock — the soundscape mixer (audio graph, per-sound volume,
// master mute) lands in T-10.
export function SoundDock() {
  return (
    <div style={dock}>
      <span style={{ fontSize: 12, color: "var(--text-faint)" }}>Soundscapes — T-10</span>
    </div>
  );
}

const dock: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  padding: "10px 16px",
  background: "var(--surface-1)",
  border: "var(--hairline)",
  borderRadius: "var(--r-lg)",
};
