export interface AuthUser {
  id: number;
  full_name: string;
  email: string;
  role: "admin" | "staff" | "reviewer";
}

export interface LoginResponse {
  token: string;
  token_type: "Bearer";
  expires_at: string;
  user: AuthUser;
}

export function saveSession(data: LoginResponse): void {
  localStorage.setItem("paa_token", data.token);
  localStorage.setItem("paa_user", JSON.stringify(data.user));
  localStorage.setItem("paa_expires_at", data.expires_at);
}

export function clearSession(): void {
  localStorage.removeItem("paa_token");
  localStorage.removeItem("paa_user");
  localStorage.removeItem("paa_expires_at");
}

export function getStoredUser(): AuthUser | null {
  const rawUser = localStorage.getItem("paa_user");

  if (!rawUser) return null;

  try {
    return JSON.parse(rawUser) as AuthUser;
  } catch {
    return null;
  }
}
