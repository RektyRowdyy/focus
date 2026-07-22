import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

// Self-hosted Inter (weights used across the app) + design tokens + base styles.
import "@fontsource/inter/300.css";
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "./theme/tokens.css";
import "./theme/global.css";

import App from "./App";
import { AuthProvider } from "./features/auth/useAuth";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </StrictMode>,
);
