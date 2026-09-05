import { useState } from "react";
import { useTranslation } from "react-i18next";

import { api } from "../../api/client";
import Button from "../objects/Button";

// The download is a one-off fetch rather than a useQuery: nothing renders the
// document, it is handed straight to the browser as a file, and caching a copy
// of everything about a person in memory is the opposite of what this is for.
export function DataExportSection() {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);

  async function download() {
    setPending(true);
    setFailed(false);
    try {
      const res = await api.get("/me/export");
      const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: "application/json" });

      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `metsatori-data-${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 0);
    } catch {
      setFailed(true);
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="space-y-2">
      <p className="text-muted text-sm">{t("pages.profile.exportExplainer")}</p>
      <Button variant="secondary" onClick={download} disabled={pending}>
        {pending ? t("pages.profile.exportWorking") : t("pages.profile.exportDownload")}
      </Button>
      {failed && (
        <p role="alert" className="text-berry-500 text-sm">
          {t("pages.profile.exportFailed")}
        </p>
      )}
    </div>
  );
}
