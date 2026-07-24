// Mirrors the backend sessions DTO (internal/sessions/dto.go).

export type SessionType = "focus" | "short" | "long";

export interface Session {
  id: number;
  startedAt: string; // RFC3339
  durationMin: number;
  type: SessionType;
  label?: string;
}

// POST /sessions body.
export interface SessionInput {
  startedAt: string; // RFC3339
  durationMin: number;
  type: SessionType;
  label?: string;
}
