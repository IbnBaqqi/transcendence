import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

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

export function categoryNames(categories: Category[]): (slug: string) => string {
  const names = new Map(flattenCategories(categories).map((c) => [c.slug, c.name]));
  return (slug: string) => names.get(slug) ?? slug;
}

export function useCategoryNames(): (slug: string) => string {
  const { data } = useCategories();
  return useMemo(() => categoryNames(data ?? []), [data]);
}

/**
 * Like useCategoryNames, but prefers a `categories.<slug>` label from the
 * active locale when one exists, falling back to the backend-provided name.
 * The backend keeps returning slugs, so this is a pure frontend lookup.
 */
export function useLocalizedCategoryNames(): (slug: string) => string {
  const { t } = useTranslation();
  const backendName = useCategoryNames();
  return useMemo(() => {
    return (slug: string) => {
      const key = `categories.${slug}`;
      const translated = t(key);
      return translated !== key ? translated : backendName(slug);
    };
  }, [backendName, t]);
}
