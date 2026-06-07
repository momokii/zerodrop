import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";
import Admin from "./pages/Admin";
import NotFound from "./pages/NotFound";

const path = window.location.pathname;
let Page: React.ComponentType;
if (path === "/admin" || path === "/admin/") {
  Page = Admin;
} else if (path === "/" || path === "") {
  Page = App;
} else {
  Page = NotFound;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Page />
  </StrictMode>
);
