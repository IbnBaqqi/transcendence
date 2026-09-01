import { useModal } from "../../providers/modalContext";
import Button from "../objects/Button.tsx";
import { useTranslation } from "react-i18next";

export function Chat() {
  const { t } = useTranslation();
  const { closeChat } = useModal();

  return (
    <div className="bg-surface-muted flex h-full flex-col">
      <div className="border-line flex items-center justify-between border-b px-4 py-3">
        <h2 className="text-foreground text-lg font-semibold">{t("chat.title")}</h2>
        <Button variant="secondary" type="button" onClick={closeChat}>
          {t("chat.close")}
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto p-4">{/* chat UI */}</div>
    </div>
  );
}
