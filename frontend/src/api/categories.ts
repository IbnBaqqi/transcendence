import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { api } from "./client";
import { keys } from "./queryKeys";
import type { Category } from "./types";

export function useCategories() {
  return useQuery({
    queryKey: keys.categories.list(),
    queryFn: async () => {
      const res = await api.get<Category[]>("/categories");
      return res.data ?? [];
    },
    staleTime: Infinity,
  });
}

export function flattenCategories(
  categories: Category[],
): { slug: string; name: string; depth: number }[] {
  return categories.flatMap((parent) => [
    { slug: parent.slug, name: parent.name, depth: 0 },
    ...parent.children.map((child) => ({ slug: child.slug, name: child.name, depth: 1 })),
  ]);
}

export function useCategoryNames(): (slug: string) => string {
  const { data } = useCategories();

  return useMemo(() => {
    const names = new Map(flattenCategories(data ?? []).map((c) => [c.slug, c.name]));
    return (slug: string) => names.get(slug) ?? slug;
  }, [data]);
}
