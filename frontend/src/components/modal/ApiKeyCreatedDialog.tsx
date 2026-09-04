import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Modal } from "./Modal";
import Button from "../objects/Button";
import type { CreatedApiKey } from "../../api/types";

type CopyState = "idle" | "copied" | "failed";

export function ApiKeyCreatedDialog({
  apiKey,
  onClose,
}: {
  apiKey: CreatedApiKey;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState<CopyState>("idle");

  async function copy() {
    try {
      // Absent outside a secure context, and can reject when permission is
      // denied - either way the key stays on screen to be copied by hand.
      await navigator.clipboard.writeText(apiKey.key);
      setCopied("copied");
    } catch {
      setCopied("failed");
    }
  }

  // variant="floating": no Escape, no backdrop dismiss. This is the only time
  // the key is ever shown, so it must not be possible to lose it by accident.
  return (
    <Modal onClose={onClose} variant="floating" className="max-w-lg">
      <div className="space-y-4 p-6">
        <h2 className="text-foreground text-section font-semibold">{t("apiKeys.created.title")}</h2>

        <p className="text-berry-500 text-sm font-medium">{t("apiKeys.created.warning")}</p>

        <div className="flex gap-2">
          <input
            readOnly
            value={apiKey.key}
            aria-label={t("apiKeys.created.keyLabel")}
            onFocus={(e) => e.currentTarget.select()}
            className="border-line bg-surface text-foreground flex-1 rounded-md border px-3 py-2 font-mono text-sm"
          />
          <Button variant="secondary" onClick={() => void copy()}>
            {copied === "copied" ? t("apiKeys.created.copied") : t("apiKeys.created.copy")}
          </Button>
        </div>

        {copied === "failed" && (
          <p role="alert" className="text-muted text-sm">
            {t("apiKeys.created.copyFailed")}
          </p>
        )}

        <p className="text-muted text-sm">{t("apiKeys.created.note", { name: apiKey.name })}</p>

        <Button variant="primary" onClick={onClose}>
          {t("apiKeys.created.done")}
        </Button>
      </div>
    </Modal>
  );
}
