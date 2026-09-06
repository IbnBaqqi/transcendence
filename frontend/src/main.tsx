import "./index.css";
import "./i18n";

import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";

import { QueryClientProvider } from "@tanstack/react-query";

import AppRouter from "./routes";
import { queryClient } from "./lib/queryClient";
import { ErrorBoundary } from "./components/layout/ErrorBoundary";

import { ModalProvider } from "./providers/ModalProvider";
import { AuthProvider } from "./providers/AuthProvider";
import { ModalRoot } from "./components/modal/ModalRoot";
import { ChatRoot } from "./components/modal/ChatRoot";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <ModalProvider>
            {/* One router for the whole tree: ChatRoot is a sibling of
                AppRouter, not a child, yet its content links to /users/:id -
                put the router above both so portalled overlays can navigate.
                See the regression test in main.test.tsx. */}
            <BrowserRouter>
              <AppRouter />
              <ChatRoot />
              <ModalRoot />
            </BrowserRouter>
          </ModalProvider>
        </AuthProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);
