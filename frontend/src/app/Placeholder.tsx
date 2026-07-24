// Shared placeholder used by not-yet-built screens.
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
