import { useTranslation } from "react-i18next";

import type { Presence } from "../../api/types";

export function PresenceIndicator({ presence }: { presence: Presence }) {
  const { t } = useTranslation();

  return (
    <span className="text-muted flex items-center gap-2 text-sm">
      <span
        aria-hidden="true"
        className={`h-2 w-2 rounded-full ${presence.is_online ? "bg-accent" : "bg-surface-soft"}`}
      />
      {presence.is_online ? t("presence.online") : t("presence.offline")}
    </span>
  );
}
