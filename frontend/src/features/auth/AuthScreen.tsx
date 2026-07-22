import { useState, type CSSProperties, type FormEvent } from "react";
import { ApiError } from "../../api/client";
import { useAuth } from "./useAuth";

// Minimal, calm auth screen: one form that toggles between login and signup.
export function AuthScreen() {
  const { login, signup } = useAuth();
  const [mode, setMode] = useState<"login" | "signup">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await (mode === "login" ? login : signup)({ email, password });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  };

  return (
    <main style={wrap}>
      <form onSubmit={submit} style={card}>
        <h1 style={brand}>Focus</h1>
        <input
          style={field}
          type="email"
          placeholder="Email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <input
          style={field}
          type="password"
          placeholder="Password"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && <p style={errText}>{error}</p>}
        <button style={submitBtn} type="submit" disabled={busy}>
          {mode === "login" ? "Log in" : "Create account"}
        </button>
        <button
          style={toggle}
          type="button"
          onClick={() => {
            setMode(mode === "login" ? "signup" : "login");
            setError(null);
          }}
        >
          {mode === "login" ? "Need an account? Sign up" : "Have an account? Log in"}
        </button>
      </form>
    </main>
  );
}

const wrap: CSSProperties = {
  minHeight: "100vh",
  display: "grid",
  placeItems: "center",
  padding: 24,
};

const card: CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 14,
  width: "100%",
  maxWidth: 320,
  padding: 28,
  background: "var(--surface-1)",
  border: "var(--hairline)",
  borderRadius: "var(--r-xl)",
};

const brand: CSSProperties = {
  fontWeight: 300,
  fontSize: 22,
  letterSpacing: "0.18em",
  textTransform: "uppercase",
  textAlign: "center",
  color: "var(--text-strong)",
  marginBottom: 6,
};

const field: CSSProperties = {
  minHeight: "var(--touch)",
  padding: "0 14px",
  background: "var(--surface-2)",
  border: "var(--hairline)",
  borderRadius: "var(--r-sm)",
  color: "var(--text-strong)",
  outline: "none",
};

const submitBtn: CSSProperties = {
  minHeight: "var(--touch)",
  background: "var(--accent-tint-strong)",
  color: "var(--text-strong)",
  border: "none",
  borderRadius: "var(--r-sm)",
  cursor: "pointer",
  transition: "background var(--dur) var(--ease)",
};

const toggle: CSSProperties = {
  background: "none",
  border: "none",
  color: "var(--text-faint)",
  fontSize: 12,
  cursor: "pointer",
};

const errText: CSSProperties = {
  color: "var(--text-muted)",
  fontSize: 13,
};
