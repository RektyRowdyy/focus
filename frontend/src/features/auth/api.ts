import { get, post } from "../../api/client";
import type { Credentials, User } from "./types";

export const authApi = {
  me: () => get<User>("/auth/me"),
  login: (c: Credentials) => post<User>("/auth/login", c),
  signup: (c: Credentials) => post<User>("/auth/signup", c),
  logout: () => post<void>("/auth/logout"),
};
