import { get, put } from "../../api/client";
import type { Settings, SettingsUpdate } from "./types";

export const settingsApi = {
  get: () => get<Settings>("/settings"),
  update: (patch: SettingsUpdate) => put<Settings>("/settings", patch),
};
