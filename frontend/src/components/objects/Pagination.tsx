import { useTranslation } from "react-i18next";

import Button from "./Button";

export function Pagination({
  page,
  totalPages,
  total,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
}) {
  const { t } = useTranslation();

  return (
    <nav aria-label={t("pagination.label")} className="flex flex-wrap items-center gap-3">
      {totalPages > 1 && (
        <>
          <Button variant="secondary" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
            {t("pagination.previous")}
          </Button>
          <span className="text-muted text-sm">{t("pagination.pageOf", { page, totalPages })}</span>
          <Button
            variant="secondary"
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            {t("pagination.next")}
          </Button>
        </>
      )}
      {/* Outside the guard: a one-page result still says how many it found. */}
      <span className="text-muted text-sm">{t("pagination.results", { count: total })}</span>
    </nav>
  );
}
