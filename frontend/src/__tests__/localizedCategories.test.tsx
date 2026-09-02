import type { ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { useLocalizedCategoryNames } from "../api/categories";
import { keys } from "../api/queryKeys";
import i18next from "../i18n";
import type { Category } from "../api/types";

const TREE: Category[] = [
  { slug: "berries", name: "Berries", children: [] },
  {
    slug: "mushrooms",
    name: "Mushrooms",
    children: [{ slug: "chanterelles", name: "Chanterelles", children: [] }],
  },
];

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient();
  client.setQueryData(keys.categories.list(), TREE);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useLocalizedCategoryNames", () => {
  beforeEach(async () => {
    if (i18next.language !== "en") await i18next.changeLanguage("en");
  });

  test("prefers a categories.<slug> label when one exists", () => {
    const { result } = renderHook(() => useLocalizedCategoryNames(), { wrapper });

    expect(result.current("mushrooms")).toBe("Mushrooms");
  });

  test("falls back to the backend name for slugs without a label", () => {
    const { result } = renderHook(() => useLocalizedCategoryNames(), { wrapper });

    expect(result.current("chanterelles")).toBe("Chanterelles");
  });

  test("falls back to the slug itself when nothing else knows it", () => {
    const { result } = renderHook(() => useLocalizedCategoryNames(), { wrapper });

    expect(result.current("truffles")).toBe("truffles");
  });

  test("labels follow the active locale", async () => {
    const { result } = renderHook(() => useLocalizedCategoryNames(), { wrapper });

    await i18next.changeLanguage("fi");
    await waitFor(() => expect(result.current("mushrooms")).toBe("Sienet"));
    expect(result.current("berries")).toBe("Marjat");
  });
});
