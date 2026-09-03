import { useTranslation } from "react-i18next";

import Button from "./Button";

type PaginationProps = {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
};

export function Pagination({ page, totalPages, onChange }: PaginationProps) {
  const { t } = useTranslation();
  if (totalPages <= 1) return null;

  return (
    <nav
      aria-label={t("common.pagination.nav")}
      className="mt-6 flex items-center justify-center gap-3 text-sm"
    >
      <Button variant="secondary" disabled={page <= 1} onClick={() => onChange(page - 1)}>
        {t("common.pagination.previous")}
      </Button>
      <span className="text-muted">{t("common.pagination.pageOf", { page, total: totalPages })}</span>
      <Button variant="secondary" disabled={page >= totalPages} onClick={() => onChange(page + 1)}>
        {t("common.pagination.next")}
      </Button>
    </nav>
  );
}
