import type { CSSProperties } from "react";
import { AppShell } from "./app/AppShell";
import { AuthScreen } from "./features/auth/AuthScreen";
import { useAuth } from "./features/auth/useAuth";

// Route guard: splash while the session resolves, then auth screen or the shell.
export default function App() {
  const { user, loading } = useAuth();

  if (loading) return <Splash />;
  return user ? <AppShell /> : <AuthScreen />;
}

function Splash() {
  return <div style={splash} aria-hidden />;
}

const splash: CSSProperties = {
  minHeight: "100vh",
  background: "var(--bg)",
};
