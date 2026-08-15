import "./index.css";

import React from "react";
import ReactDOM from "react-dom/client";

import { QueryClientProvider } from "@tanstack/react-query";

import AppRouter from "./routes";
import { queryClient } from "./lib/queryClient";
import { ErrorBoundary } from "./components/ErrorBoundary";

import { ModalProvider } from "./providers/ModalProvider";
import { ModalRoot } from "./components/modal/ModalRoot";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <ModalProvider>
          <AppRouter />
          <ModalRoot />
        </ModalProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);
