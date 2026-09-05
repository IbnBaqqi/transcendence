// index.css defines colour ROLES and swaps their values in dark mode; a class
// naming a palette entry directly cannot swap, which is how every error
// message in the app came to sit at 3.55:1 on the dark ground - under the
// 4.5:1 floor - while passing in light. CLAUDE.md states the rule; this
// enforces it. Same source scan as iconSprite.test.ts and typeScale.test.ts.
const sources = import.meta.glob("../{components,pages}/**/*.tsx", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const PALETTE = /\b(?:text|bg|border|ring|fill|stroke|from|via|to)-(?:brand|berry)-\d{2,3}\b/g;

function rawPaletteUses(files: Record<string, string>) {
  return Object.entries(files).flatMap(([path, code]) =>
    [...code.matchAll(PALETTE)].map((match) => `${path}: ${match[0]}`),
  );
}

test("the glob actually found the components", () => {
  // Without this the check below passes vacuously on an empty scan.
  expect(Object.keys(sources).length).toBeGreaterThan(20);
});

test("components name colour roles, not palette entries", () => {
  expect(rawPaletteUses(sources)).toEqual([]);
});

// The positive control: the check is a filter that finds nothing, and a filter
// whose regex has stopped matching finds nothing either.
test("the check catches a raw palette class", () => {
  expect(rawPaletteUses({ "x.tsx": '<p className="text-berry-500">t</p>' })).toHaveLength(1);
});
