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
          {/* min: a stale link can ask for a page past the end, and the server
              answers that with an empty page. Step back to the last real one. */}
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
