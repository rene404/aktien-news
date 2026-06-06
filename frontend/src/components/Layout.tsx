import { Link, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <div className="app">
      <header className="topbar">
        <Link to="/" className="brand">
          📈 Aktien News
        </Link>
        <nav>
          <Link to="/">Search</Link>
          {user && <Link to="/watchlists">Watchlists</Link>}
          {user?.role === "admin" && <Link to="/admin/feeds">Feeds</Link>}
          {user?.role === "admin" && <Link to="/admin/review">Review</Link>}
        </nav>
        <div className="spacer" />
        {user ? (
          <div className="user">
            <span>{user.email}</span>
            <button
              onClick={() => {
                logout();
                navigate("/login");
              }}
            >
              Logout
            </button>
          </div>
        ) : (
          <div className="user">
            <Link to="/login">Login</Link>
            <Link to="/register">Register</Link>
          </div>
        )}
      </header>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
