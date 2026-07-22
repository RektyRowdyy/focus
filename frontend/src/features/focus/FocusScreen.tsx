// Placeholder — the timer engine + focus session UI land in T-08.
export function FocusScreen() {
  return <Placeholder label="Focus" note="Timer + session controls — T-08" />;
}

// Shared tiny placeholder used by the scaffold screens.
export function Placeholder({ label, note }: { label: string; note: string }) {
  return (
    <div
      style={{
        display: "grid",
        placeItems: "center",
        gap: 8,
        minHeight: "60vh",
        textAlign: "center",
      }}
    >
      <span
        style={{
          fontWeight: 300,
          fontSize: 28,
          letterSpacing: "0.02em",
          color: "var(--text-strong)",
        }}
      >
        {label}
      </span>
      <span style={{ fontSize: 12, color: "var(--text-faint)" }}>{note}</span>
    </div>
  );
}
