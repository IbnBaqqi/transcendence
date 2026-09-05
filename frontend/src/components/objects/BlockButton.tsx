import { useState } from "react";
import { useTranslation } from "react-i18next";

import Button from "./Button";
import { isApiError } from "../../api/client";
import { useBlock, useBlocks, useUnblock } from "../../api/blocks";
import { useAuth } from "../../hooks/useAuth";

export function BlockButton({ userId }: { userId: string }) {
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useTranslation();
  // Local state rather than the mutation's own isError, unlike
  // BlockedUsersSection: there are two mutations here, so reading isError would
  // mean OR-ing them and picking whichever error is the live one.
  const [error, setError] = useState<string | null>(null);

  const { data: blocks, isPending: blocksPending } = useBlocks({ enabled: Boolean(user) });
  const block = useBlock();
  const unblock = useUnblock();

  if (authLoading || !user || user.id === userId) return null;

  const isBlocked = blocks?.some((b) => b.id === userId) ?? false;
  const busy = block.isPending || unblock.isPending;

  async function toggle() {
    setError(null);
    try {
      await (isBlocked ? unblock : block).mutateAsync(userId);
    } catch (err) {
      setError(isApiError(err) ? err.message : t("common.somethingWentWrong"));
    }
  }

  return (
    <div className="space-y-2">
      <Button variant="secondary" disabled={busy || blocksPending} onClick={() => void toggle()}>
        {busy ? t("blocks.updating") : isBlocked ? t("blocks.unblock") : t("blocks.block")}
      </Button>
      {error && (
        <p role="alert" className="text-danger text-sm">
          {error}
        </p>
      )}
    </div>
  );
}
