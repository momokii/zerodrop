import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";
import Admin from "./pages/Admin";

// Simple URL-based routing — no react-router needed for two pages
const isAdmin = window.location.pathname === "/admin" || window.location.pathname === "/admin/";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    {isAdmin ? <Admin /> : <App />}
  </StrictMode>
);
