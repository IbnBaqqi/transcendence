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
      {/* A stale link can ask for a page past the end: keep the controls, since
          they are the only way back, and step to the last real page. */}
      {(totalPages > 1 || page > totalPages) && (
        <>
          <Button
            variant="secondary"
            disabled={page <= 1}
            onClick={() => onPageChange(Math.min(page - 1, totalPages))}
          >
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
      <span className="text-muted text-sm">{t("pagination.results", { count: total })}</span>
    </nav>
  );
}
