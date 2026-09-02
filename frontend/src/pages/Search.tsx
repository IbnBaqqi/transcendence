// stub for #25
//
// no useSearchParams() here yet - importing it without using it would fail
// the build. whoever take #25 adds it: it's the hook that reads and writes
// the query string (?category=berries&min_price=5), which is how filter
// state gets shared in a URL instead of trapped in component state.
import { useTranslation } from "react-i18next";
export default function Search() {
  const { t } = useTranslation();
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{t("pages.search.title")}</h1>
      <p className="text-muted mt-2">
        {/* TODO(#25): filter panel (category, price range, location, rating),
					active filters as dismissible chips, state synced to URL params
					via useSearchParams. rating filter needs #16. */}
        {t("pages.search.stub")}
      </p>
    </div>
  );
}
