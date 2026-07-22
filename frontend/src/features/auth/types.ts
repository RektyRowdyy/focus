// Mirrors the backend auth DTOs (internal/auth/dto.go).

export interface User {
  id: number;
  email: string;
}

export interface Credentials {
  email: string;
  password: string;
}
