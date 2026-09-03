import { useSearchParams } from "react-router-dom";

import { filtersToParams, parseFilters, withFilters } from "../lib/searchFilters";

// For pages whose only URL state is the page number. Search owns the whole
// filter set and writes it itself.
export function usePageParam(): [number, (page: number) => void] {
  const [params, setParams] = useSearchParams();
  const filters = parseFilters(params);

  return [filters.page, (page) => setParams(filtersToParams(withFilters(filters, { page })))];
}
