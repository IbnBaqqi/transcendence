import en from "../i18n/locales/en/translation.json";
import fi from "../i18n/locales/fi/translation.json";
import sv from "../i18n/locales/sv/translation.json";

function flattenKeys(record: Record<string, unknown>, prefix = ""): string[] {
  return Object.entries(record).flatMap(([key, value]) =>
    typeof value === "object" && value !== null
      ? flattenKeys(value as Record<string, unknown>, `${prefix}${key}.`)
      : [`${prefix}${key}`],
  );
}

describe("translation files", () => {
  test("every locale defines exactly the same keys as English", () => {
    const english = flattenKeys(en).sort();
    const finnish = flattenKeys(fi).sort();
    const swedish = flattenKeys(sv).sort();

    expect(finnish).toEqual(english);
    expect(swedish).toEqual(english);
  });
});
