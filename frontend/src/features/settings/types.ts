// Mirrors the backend user_settings DTO (internal/settings/dto.go).

export interface Settings {
  focusMin: number;
  shortMin: number;
  longMin: number;
  longInterval: number;
  autoBreaks: boolean;
  autoFocus: boolean;
  chime: boolean;
  theme: "dark" | "light";
  masterMute: boolean;
}

// Partial update body for PUT /settings (only sent fields change).
export type SettingsUpdate = Partial<Settings>;

// Fallback used before the server responds / when offline. Matches the DB defaults.
export const DEFAULT_SETTINGS: Settings = {
  focusMin: 25,
  shortMin: 5,
  longMin: 15,
  longInterval: 4,
  autoBreaks: false,
  autoFocus: false,
  chime: true,
  theme: "dark",
  masterMute: false,
};
