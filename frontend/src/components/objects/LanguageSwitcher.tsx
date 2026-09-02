import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import { supportedLngs, type Locale } from "../../i18n";

// flag emojis render regionally; on platforms without them they degrade to
// letter codes, which is acceptable for the collapsed "just a flag" affordance
const FLAGS: Record<Locale, string> = {
  en: "🇬🇧",
  fi: "🇫🇮",
  sv: "🇸🇪",
};

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  const current = (i18n.language?.split("-")[0] ?? "en") as Locale;
  const active = supportedLngs.includes(current) ? current : "en";

  // Close when the user clicks anywhere outside, or presses Escape.
  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t("language.label")}
        aria-expanded={open}
        className="text-muted hover:text-foreground translate-y-px cursor-pointer text-2xl leading-none focus:outline-none"
      >
        <span aria-hidden="true">{FLAGS[active]}</span>
      </button>

      {open && (
        <div
          role="group"
          aria-label={t("language.label")}
          className="border-line bg-surface absolute top-full right-0 z-20 mt-2 rounded border shadow-lg"
        >
          {supportedLngs.map((lng) => (
            <button
              key={lng}
              type="button"
              aria-current={lng === active ? "true" : undefined}
              onClick={() => {
                if (lng !== active) void i18n.changeLanguage(lng);
                setOpen(false);
                triggerRef.current?.focus();
              }}
              className={`flex w-full items-center gap-2 px-3 py-2 text-left text-sm ${
                lng === active ? "text-accent font-medium" : "text-foreground hover:bg-surface-soft"
              }`}
            >
              <span aria-hidden="true" className="text-base leading-none">
                {FLAGS[lng]}
              </span>
              {t(`language.${lng}`)}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
