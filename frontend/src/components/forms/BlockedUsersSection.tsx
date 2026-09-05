import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { useBlocks, useUnblock } from "../../api/blocks";
import { isApiError } from "../../api/client";
import Button from "../objects/Button";
import { Skeleton } from "../objects/Skeleton";

export function BlockedUsersSection() {
  const { t } = useTranslation();
  const { data: blocks, isPending, isError, error, refetch } = useBlocks();
  const unblock = useUnblock();

  if (isPending) return <Skeleton className="h-16 w-full" />;

  if (isError) {
    return (
      <Skeleton
        variant="error"
        className="h-16 w-full"
        message={isApiError(error) ? error.message : t("blocks.listError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (blocks.length === 0) {
    return <p className="text-muted text-sm">{t("blocks.listEmpty")}</p>;
  }

  return (
    <>
      <ul className="divide-line divide-y">
        {blocks.map((blocked) => (
          <li key={blocked.id} className="flex items-center justify-between gap-3 py-2">
            <Link to={`/users/${blocked.id}`} className="text-foreground hover:underline">
              {blocked.username}
            </Link>
            <Button
              variant="secondary"
              // One mutation instance serves the whole list, so isPending alone
              // would disable and relabel every row while one is in flight.
              disabled={unblock.isPending && unblock.variables === blocked.id}
              onClick={() => unblock.mutate(blocked.id)}
            >
              {unblock.isPending && unblock.variables === blocked.id
                ? t("blocks.updating")
                : t("blocks.unblock")}
            </Button>
          </li>
        ))}
      </ul>
      {unblock.isError && (
        <p role="alert" className="text-danger text-sm">
          {isApiError(unblock.error) ? unblock.error.message : t("common.somethingWentWrong")}
        </p>
      )}
    </>
  );
}
