import { useState, type CSSProperties } from "react";
import { useAuth } from "../features/auth/useAuth";
import { FocusScreen } from "../features/focus/FocusScreen";
import { InsightsScreen } from "../features/insights/InsightsScreen";
import { SettingsScreen } from "../features/settings/SettingsScreen";
import { SoundDock } from "../features/soundscapes/SoundDock";

type View = "focus" | "insights" | "settings";

const NAV: { id: View; label: string }[] = [
  { id: "focus", label: "Focus" },
  { id: "insights", label: "Insights" },
  { id: "settings", label: "Settings" },
];

// State-based shell (no router): a nav switches the active view; the
// soundscape dock persists across views. Responsive polish is T-13.
export function AppShell() {
  const { user, logout } = useAuth();
  const [view, setView] = useState<View>("focus");

  return (
    <div style={shell}>
      <header style={header}>
        <nav style={{ display: "flex", gap: 4 }}>
          {NAV.map((n) => (
            <button
              key={n.id}
              onClick={() => setView(n.id)}
              style={navBtn(view === n.id)}
            >
              {n.label}
            </button>
          ))}
        </nav>
        <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
          <span style={{ fontSize: 12, color: "var(--text-faint)" }}>{user?.email}</span>
          <button onClick={logout} style={logoutBtn}>
            Log out
          </button>
        </div>
      </header>

      <main style={content}>
        {view === "focus" && <FocusScreen />}
        {view === "insights" && <InsightsScreen />}
        {view === "settings" && <SettingsScreen />}
      </main>

      <footer style={footer}>
        <SoundDock />
      </footer>
    </div>
  );
}

const shell: CSSProperties = {
  minHeight: "100vh",
  display: "grid",
  gridTemplateRows: "auto 1fr auto",
};

const header: CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  padding: "14px 20px",
  borderBottom: "var(--hairline)",
};

const navBtn = (active: boolean): CSSProperties => ({
  minHeight: 36,
  padding: "0 14px",
  background: active ? "var(--accent-tint)" : "transparent",
  color: active ? "var(--text-strong)" : "var(--text-faint)",
  border: "none",
  borderRadius: "var(--r-sm)",
  cursor: "pointer",
  transition: "color var(--dur) var(--ease), background var(--dur) var(--ease)",
});

const logoutBtn: CSSProperties = {
  minHeight: 36,
  padding: "0 12px",
  background: "transparent",
  color: "var(--text-faint)",
  border: "var(--hairline)",
  borderRadius: "var(--r-sm)",
  cursor: "pointer",
};

const content: CSSProperties = {
  padding: 24,
};

const footer: CSSProperties = {
  display: "flex",
  justifyContent: "flex-start",
  padding: "14px 20px",
  borderTop: "var(--hairline)",
};
