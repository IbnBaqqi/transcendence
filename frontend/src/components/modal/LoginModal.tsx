import { useModal } from "../../providers/modalContext";
import { LoginSection } from "../forms/LoginSection";
import Button from "../objects/Button.tsx";
import { useTranslation } from "react-i18next";

export function LoginModal() {
  const { t } = useTranslation();
  const { closeModal, openModal } = useModal();

  return (
    <div className="space-y-2 p-6">
      <h2 className="text-section mb-4 font-semibold">{t("modal.login.title")}</h2>
      <LoginSection onClose={closeModal} />
      <div className="">
        <span>{t("modal.login.notRegistered")}</span>
        <Button variant="tertiary" onClick={() => openModal("register")}>
          {t("common.register")}
        </Button>
      </div>
    </div>
  );
}
