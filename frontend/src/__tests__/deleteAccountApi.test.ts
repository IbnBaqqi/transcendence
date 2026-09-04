import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { api } from "../api/client";
import { useDeleteAccount } from "../api/profile";

vi.mock("../api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/client")>()),
  api: { delete: vi.fn().mockResolvedValue({ data: undefined }) },
}));

// axios sends a DELETE body only under `data`. Passing the object directly
// compiles, is read as request config, and silently sends nothing - so the
// server sees no username and answers 400 for a name typed correctly, which
// reads from the screen as "the API is broken".
//
// DeleteAccountSection's own tests mock this hook, so they cannot see it.
test("the confirmation travels as a body, not as request config", async () => {
  const { result } = renderHook(() => useDeleteAccount(), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(
        QueryClientProvider,
        { client: new QueryClient({ defaultOptions: { queries: { retry: false } } }) },
        children,
      ),
  });

  await result.current.mutateAsync("forager");

  expect(api.delete).toHaveBeenCalledWith("/me", { data: { username: "forager" } });
});
