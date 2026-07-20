import "./index.css";

import React from "react";
import ReactDOM from "react-dom/client";

import { QueryClientProvider } from "@tanstack/react-query";

import AppRouter from "./routes";
import { queryClient } from "./lib/queryClient";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    {/* anything rendered inside can now use React Query hooks and will share this one client's cache */}
    <QueryClientProvider client={queryClient}>
      <AppRouter />
    </QueryClientProvider>
  </React.StrictMode>,
);
