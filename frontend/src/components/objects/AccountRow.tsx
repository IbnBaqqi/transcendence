import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useUserHistory } from "../../api/adminUsers";
import { AccountActions } from "./AccountActions";
import type { AdminUser, UserAction } from "../../api/types";

const STATUS_STYLES: Record<AdminUser["status"], string> = {
  active: "text-foreground",
  suspended: "text-berry-500",
  // Deleted rows carry no information but do take up space, so they stay
  // legible and quiet rather than being filtered out of the default view.
  deleted: "text-muted",
};

function HistoryLine({ entry }: { entry: UserAction }) {
  const { t } = useTranslation();

  return (
    <li className="py-2">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-foreground text-sm font-medium">
          {t(`adminUsers.action.${entry.action}`)}
        </span>
        <span className="text-muted shrink-0 text-xs">
          {new Date(entry.created_at).toLocaleDateString()}
        </span>
      </div>
      {/* Empty only for a reinstatement, the one action needing no reason. */}
      {entry.note !== "" && <p className="text-muted mt-0.5 text-sm">{entry.note}</p>}
      {entry.moderator_id === null && (
        <p className="text-muted mt-0.5 text-xs italic">{t("adminUsers.deletedModerator")}</p>
      )}
    </li>
  );
}

export function AccountRow({ account }: { account: AdminUser }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const deleted = account.status === "deleted";

  // A collapsed row passes "", leaving the query disabled: opening one row is
  // what costs a request, not rendering the list.
  const { data: history, isError } = useUserHistory(expanded ? account.id : "");

  return (
    <li className="border-line bg-surface rounded-lg border p-3">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          {/* deleted-<id> is anonymisation, not a name: showing it would put an
              id on screen where a person used to be. */}
          <p className={`font-medium ${deleted ? "text-muted italic" : "text-foreground"}`}>
            {deleted ? t("adminUsers.deletedUser") : account.username}
          </p>
          {!deleted && <p className="text-muted text-sm">{account.email}</p>}
          {account.suspension_reason && (
            <p className="text-muted mt-1 text-sm">
              {t("adminUsers.suspendedFor", { reason: account.suspension_reason })}
            </p>
          )}
        </div>

        {/* No shrink-0: it made this column refuse to narrow, so the button
            row inside it never ran out of width and never wrapped. Both halves
            are needed - flex-wrap on the buttons is inert while the column
            they sit in cannot give any width back. */}
        <div className="flex flex-col items-end gap-1 text-sm">
          <span className={STATUS_STYLES[account.status]}>
            {t(`adminUsers.status.${account.status}`)}
          </span>
          <span className="text-muted text-xs">{t(`adminUsers.role.${account.role}`)}</span>
          {/* A moderation signal, not presence: this ignores the user's own
              show_online_status setting, so it must never read as "online". */}
          <span className="text-muted text-xs">
            {account.last_seen_at
              ? t("adminUsers.lastSeen", {
                  date: new Date(account.last_seen_at).toLocaleDateString(),
                })
              : t("adminUsers.neverSeen")}
          </span>
          <AccountActions account={account} />
          {/* Offered for a deleted account too, unlike the actions: keeping the
              history is the point of anonymising the row rather than dropping it. */}
          <button
            type="button"
            onClick={() => setExpanded((value) => !value)}
            aria-expanded={expanded}
            className="text-accent cursor-pointer text-xs"
          >
            {expanded ? t("adminUsers.hideHistory") : t("adminUsers.showHistory")}
          </button>
        </div>
      </div>

      {expanded && (
        <div className="border-line mt-3 border-t pt-3">
          <h3 className="text-foreground text-secondary font-bold">
            {t("adminUsers.historyTitle")}
          </h3>
          {history && history.length > 0 ? (
            // Server order, newest first. Nothing here is edited or deleted, so
            // an account suspended and reinstated twice shows all four rows.
            <ul className="divide-line divide-y">
              {history.map((entry) => (
                <HistoryLine key={entry.id} entry={entry} />
              ))}
            </ul>
          ) : (
            <p className="text-muted mt-1 text-sm">
              {history
                ? t("adminUsers.historyEmpty")
                : isError
                  ? t("adminUsers.historyError")
                  : t("common.loading")}
            </p>
          )}
        </div>
      )}
    </li>
  );
}
