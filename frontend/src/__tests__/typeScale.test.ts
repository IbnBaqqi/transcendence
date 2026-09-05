// The type scale in index.css names three heading ranks, and nothing enforced
// that a heading picks one - which is how three card titles ended up with no
// size at all, inheriting body size and outranking their own body by weight
// alone. Imported through the glob rather than read from disk, like
// iconSprite.test.ts: the app tsconfig has no node types.
const sources = import.meta.glob("../{components,pages}/**/*.tsx", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const ROLES = ["text-page-title", "text-section", "text-item-title"];

// JSX comments are stripped first, on the same reasoning as the sprite test:
// a commented-out heading renders nothing, so holding it to the rule would
// only ban editing dead code.
function bareHeadings(files: Record<string, string>) {
  return Object.entries(files).flatMap(([path, code]) =>
    [...code.replace(/\{\/\*[\s\S]*?\*\/\}/g, "").matchAll(/<h[1-6][^>]*>/g)]
      .filter((match) => !ROLES.some((role) => match[0].includes(role)))
      .map((match) => `${path}: ${match[0].replace(/\s+/g, " ")}`),
  );
}

test("the glob actually found the components", () => {
  // Without this the check below passes vacuously on an empty scan.
  expect(Object.keys(sources).length).toBeGreaterThan(20);
});

test("every heading names its rank in the type scale", () => {
  expect(bareHeadings(sources)).toEqual([]);
});

// The positive control: the check above is a filter that finds nothing, and
// a filter whose regex has stopped matching finds nothing either.
test("the check catches a heading with no role", () => {
  expect(
    bareHeadings({ "x.tsx": '<h2 className="text-foreground font-bold">t</h2>' }),
  ).toHaveLength(1);
});
