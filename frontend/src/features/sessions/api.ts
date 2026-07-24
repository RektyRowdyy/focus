import { get, post } from "../../api/client";
import type { Session, SessionInput } from "./types";

export const sessionsApi = {
  create: (input: SessionInput) => post<Session>("/sessions", input),
  list: (query = "") => get<Session[]>(`/sessions${query}`),
};
