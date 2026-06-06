import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { RequireAdmin, RequireAuth } from "./Guards";
import { useAuth } from "../auth/AuthContext";
import type { UserOut } from "../api/types";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));

const mockedUseAuth = vi.mocked(useAuth);

type AuthShape = {
  user: UserOut | null;
  loading: boolean;
};

function setAuth(state: AuthShape) {
  mockedUseAuth.mockReturnValue({
    ...state,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  });
}

function renderGuarded(guard: "auth" | "admin") {
  const Guard = guard === "auth" ? RequireAuth : RequireAdmin;
  return render(
    <MemoryRouter initialEntries={["/protected"]}>
      <Routes>
        <Route
          path="/protected"
          element={
            <Guard>
              <div>secret content</div>
            </Guard>
          }
        />
        <Route path="/login" element={<div>login page</div>} />
        <Route path="/" element={<div>home page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

const admin: UserOut = { id: "1", email: "a@b.com", role: "admin" };
const regular: UserOut = { id: "2", email: "u@b.com", role: "user" };

beforeEach(() => {
  mockedUseAuth.mockReset();
});

describe("RequireAuth", () => {
  it("shows a loading state while auth resolves", () => {
    setAuth({ user: null, loading: true });
    renderGuarded("auth");
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
    expect(screen.queryByText("secret content")).not.toBeInTheDocument();
  });

  it("redirects to /login when unauthenticated", () => {
    setAuth({ user: null, loading: false });
    renderGuarded("auth");
    expect(screen.getByText("login page")).toBeInTheDocument();
    expect(screen.queryByText("secret content")).not.toBeInTheDocument();
  });

  it("renders children when authenticated", () => {
    setAuth({ user: regular, loading: false });
    renderGuarded("auth");
    expect(screen.getByText("secret content")).toBeInTheDocument();
  });
});

describe("RequireAdmin", () => {
  it("redirects to /login when unauthenticated", () => {
    setAuth({ user: null, loading: false });
    renderGuarded("admin");
    expect(screen.getByText("login page")).toBeInTheDocument();
  });

  it("redirects a non-admin user to home", () => {
    setAuth({ user: regular, loading: false });
    renderGuarded("admin");
    expect(screen.getByText("home page")).toBeInTheDocument();
    expect(screen.queryByText("secret content")).not.toBeInTheDocument();
  });

  it("renders children for an admin", () => {
    setAuth({ user: admin, loading: false });
    renderGuarded("admin");
    expect(screen.getByText("secret content")).toBeInTheDocument();
  });
});
