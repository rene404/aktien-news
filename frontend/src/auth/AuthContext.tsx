import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api, tokenStore } from "../api/client";
import type { TokenResponse, UserOut } from "../api/types";

interface AuthState {
  user: UserOut | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserOut | null>(null);
  const [loading, setLoading] = useState(true);

  async function loadMe() {
    if (!tokenStore.access) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const resp = await api.get<UserOut>("/auth/me");
      setUser(resp.data);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadMe();
  }, []);

  async function login(email: string, password: string) {
    const resp = await api.post<TokenResponse>("/auth/login", { email, password });
    tokenStore.set(resp.data.access_token, resp.data.refresh_token);
    await loadMe();
  }

  async function register(email: string, password: string) {
    await api.post("/auth/register", { email, password });
    await login(email, password);
  }

  function logout() {
    tokenStore.clear();
    setUser(null);
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
