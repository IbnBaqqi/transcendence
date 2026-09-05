import sprite from "../../public/icons.svg?raw";

// The sprite and the components are two halves of a contract nothing else
// checks: a <use href="/icons.svg#x"> whose symbol does not exist renders
// silently blank. jsdom never fetches the sprite, so no component test can
// catch it - this is how the brand mark shipped invisible in Firefox.
//
// Imported rather than read from disk, like translationParity.test.ts: the app
// tsconfig has no node types, and vite resolves ?raw and glob at build time.
const sources = import.meta.glob("../{components,pages,providers,hooks}/**/*.tsx", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// Comments are stripped first. A commented-out <symbol> is not rendered by a
// browser, so counting it as defined would let a <use> pointing at one pass
// the check below while the icon draws nothing - the exact failure this file
// exists to catch. It also lets an icon be staged in the sprite ahead of the
// change that uses it, without tripping the unused check.
const live = sprite.replace(/<!--[\s\S]*?-->/g, "");
const defined = new Set([...live.matchAll(/<symbol id="([^"]+)"/g)].map((m) => m[1]));
const referenced = new Set(
  Object.values(sources).flatMap((code) =>
    [...code.matchAll(/\/icons\.svg#([a-zA-Z0-9-]+)/g)].map((m) => m[1]),
  ),
);

test("the glob actually found the components", () => {
  // Without this the two set comparisons below pass vacuously on an empty scan.
  expect(Object.keys(sources).length).toBeGreaterThan(20);
  expect(referenced.size).toBeGreaterThan(0);
});

test("every icon a component asks for exists in the sprite", () => {
  expect([...referenced].filter((id) => !defined.has(id))).toEqual([]);
});

test("a commented-out symbol does not count as defined", () => {
  // Staged icons live in the sprite as comments. If this ever passed by
  // accident, a reference to a staged icon would look resolved and render
  // blank.
  expect(sprite).toContain("profile-icon");
  expect(defined.has("profile-icon")).toBe(false);
});

// The other direction: #232 deleted six symbols nothing referenced, by hand.
test("every symbol in the sprite is referenced by a component", () => {
  expect([...defined].filter((id) => !referenced.has(id))).toEqual([]);
});

// An external <use> with no fragment asks for the target document's root
// element, which Firefox does not render.
test("no component references an external svg without a fragment", () => {
  const bare = Object.entries(sources)
    .filter(([, code]) => /<use\s+href="\/[^"#]+\.svg"/.test(code))
    .map(([path]) => path);
  expect(bare).toEqual([]);
});
