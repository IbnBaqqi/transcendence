import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useCategories, useLocalizedCategoryNames } from "../../api/categories";
import { useAuth } from "../../hooks/useAuth";

// Plain text links under the listing grid. The app's navigation is otherwise
// wordless icons and a dropdown, and the categories are reachable only by
// working the filter form on /search - so this is the only place either the
// user or a crawler is told in words what the site holds (#218).
export function HomeDirectory() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { data: categories } = useCategories();
  const categoryName = useLocalizedCategoryNames();

  // Top level only: the children are narrower than a directory wants, and this
  // is already the longest column.
  const topLevel = categories ?? [];

  return (
    <nav aria-label={t("pages.home.directory.label")} className="border-line mt-12 border-t pt-8">
      <div className="grid gap-8 sm:grid-cols-3">
        {topLevel.length > 0 && (
          <Column title={t("pages.home.directory.categories")}>
            {topLevel.map((category) => (
              // Search parses its filters straight from the URL, so these are
              // ordinary links rather than anything the filter form has to run.
              <DirectoryLink key={category.slug} to={`/search?category=${category.slug}`}>
                {categoryName(category.slug)}
              </DirectoryLink>
            ))}
          </Column>
        )}

        <Column title={t("pages.home.directory.browse")}>
          <DirectoryLink to="/search">{t("nav.search")}</DirectoryLink>
          <DirectoryLink to="/addlisting">{t("nav.addListing")}</DirectoryLink>
        </Column>

        {user && (
          <Column title={t("pages.home.directory.account")}>
            <DirectoryLink to="/dashboard">{t("pages.dashboard.title")}</DirectoryLink>
            <DirectoryLink to="/orders">{t("nav.orders")}</DirectoryLink>
            <DirectoryLink to="/following">{t("nav.following")}</DirectoryLink>
            <DirectoryLink to="/notifications">{t("nav.notifications")}</DirectoryLink>
            <DirectoryLink to="/profile">{t("nav.profile")}</DirectoryLink>
          </Column>
        )}
      </div>
    </nav>
  );
}

function Column({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h2 className="text-foreground text-item-title font-semibold">{title}</h2>
      <ul className="space-y-1">{children}</ul>
    </div>
  );
}

function DirectoryLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <li>
      <Link to={to} className="text-muted hover:text-foreground text-sm hover:underline">
        {children}
      </Link>
    </li>
  );
}
