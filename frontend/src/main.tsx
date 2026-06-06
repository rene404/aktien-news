import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import "./index.css";
import { AuthProvider } from "./auth/AuthContext";
import { RequireAdmin, RequireAuth } from "./components/Guards";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Search from "./pages/Search";
import CompanyNews from "./pages/CompanyNews";
import WatchlistPage from "./pages/WatchlistPage";
import Feeds from "./pages/admin/Feeds";
import Review from "./pages/admin/Review";

const queryClient = new QueryClient();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Search />} />
              <Route path="stocks/:stockId" element={<CompanyNews />} />
              <Route path="login" element={<Login />} />
              <Route path="register" element={<Register />} />
              <Route
                path="watchlists"
                element={
                  <RequireAuth>
                    <WatchlistPage />
                  </RequireAuth>
                }
              />
              <Route
                path="admin/feeds"
                element={
                  <RequireAdmin>
                    <Feeds />
                  </RequireAdmin>
                }
              />
              <Route
                path="admin/review"
                element={
                  <RequireAdmin>
                    <Review />
                  </RequireAdmin>
                }
              />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
