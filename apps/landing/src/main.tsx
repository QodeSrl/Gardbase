import React from "react";
import ReactDOM from "react-dom/client";
import App from "@/components/App";
import "@/index.css";
import { BrowserRouter } from "react-router-dom";
import { ThemeProvider } from "@/lib/ThemeProvider";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ThemeProvider>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ThemeProvider>
  </React.StrictMode>
);
