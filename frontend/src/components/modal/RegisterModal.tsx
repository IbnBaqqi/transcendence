import { useModal } from "../../providers/modalContext";
import { RegisterSection } from "../forms/RegisterSection";
import Button from "../objects/Button.tsx";
import { useTranslation } from "react-i18next";

export function RegisterModal() {
  const { t } = useTranslation();
  const { closeModal, openModal } = useModal();

  return (
    <div className="space-y-2 p-6">
      <h2 className="text-section mb-4 font-semibold">{t("modal.register.title")}</h2>
      <RegisterSection onClose={closeModal} />
      <div>
        <span>{t("modal.register.alreadyHaveAccount")}</span>
        <Button variant="tertiary" onClick={() => openModal("login")}>
          {t("common.login")}
        </Button>
      </div>
    </div>
  );
}
