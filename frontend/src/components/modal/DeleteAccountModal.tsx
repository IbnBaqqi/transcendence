import { useModal } from "../../providers/modalContext";
import { DeleteAccountSection } from "../forms/DeleteAccountSection";
import { useTranslation } from "react-i18next";

export function DeleteAccountModal() {
  const { t } = useTranslation();
  const { closeModal } = useModal();

  return (
    <div className="p-6">
      <h2 className="text-section mb-4 font-semibold">{t("modal.deleteAccount.title")}</h2>
      <DeleteAccountSection onClose={closeModal} />
    </div>
  );
}
