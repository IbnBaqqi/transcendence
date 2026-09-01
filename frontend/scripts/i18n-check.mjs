#!/usr/bin/env node
// Guard: catches hardcoded UI strings in pages/ and components/ that bypass
// the i18n layer. It parses each .tsx file with the TypeScript compiler and
// flags
//   1. JSX text nodes that read as prose (letters + a space)
//   2. UI attribute string literals (label="", placeholder="", ...)
//   3. default prop values in signatures (`retryLabel = "Try again",`)
// so brand names and single-word tokens are intentionally ignored.
//
// Run with: npm run i18n:check

import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import * as ts from "typescript";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const resolve = require.resolve("typescript");

const SRC = join(import.meta.dirname, "..", "src");
const SCAN_DIRS = ["pages", "components"];
const UI_ATTRS = new Set([
  "label",
  "placeholder",
  "aria-label",
  "helperText",
  "emptyMessage",
  "title",
]);

function walk(dir) {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    return statSync(path).isDirectory() ? walk(path) : path;
  });
}

const files = SCAN_DIRS.flatMap((dir) => walk(join(SRC, dir))).filter((f) => f.endsWith(".tsx"));

function isProse(text) {
  return /[A-Za-z]/.test(text) && /\s/.test(text) && !/^[?—–‑]+$/.test(text);
}

const issues = [];

for (const file of files) {
  const source = readFileSync(file, "utf8");
  const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const report = (pos, text, kind) =>
    issues.push({ file, line: sf.getLineAndCharacterOfPosition(pos).line + 1, text, kind });

  // 1. Real JSX text nodes, via the TS AST (no regex heuristics).
  const visit = (node) => {
    if (ts.isJsxText(node)) {
      const text = node.getText(sf).trim();
      if (isProse(text)) report(node.getStart(sf), text, "jsx-text");
    } else if (
      ts.isJsxElement(node) ||
      ts.isJsxFragment(node) ||
      ts.isJsxSelfClosingElement(node)
    ) {
      // children already visited via forEachChild; nothing special needed
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);

  // 2. UI attribute string literals (label="...", placeholder="...", ...)
  // Metadata text like className="..." is in the AST as string literals too,
  // so walk attributes rather than regex, filtering by attr name.
  const attrVisit = (node) => {
    if (ts.isJsxAttribute(node) && node.initializer && ts.isStringLiteral(node.initializer)) {
      const name = node.name.getText(sf);
      const value = node.initializer.text;
      if (
        UI_ATTRS.has(name) &&
        /[A-Za-z]/.test(value) &&
        (/\s/.test(value) || /^[A-Z]/.test(value))
      ) {
        report(node.initializer.getStart(sf), value, "attr");
      }
    }
    ts.forEachChild(node, attrVisit);
  };
  attrVisit(sf);

  // 3. Default prop values in signatures (`retryLabel = "Try again",`).
  // Only flag prose-like values: CSS class strings and SVG geometry (which are
  // lowercase, colon/bracket/slash-heavy) that dominate attribute values.
  for (const match of source.matchAll(/([A-Za-z]+)\s*=\s*"([^"]+)"\s*,?$/gm)) {
    const value = match[2];
    if (!/[A-Za-z]/.test(value)) continue;
    if (!/\s/.test(value)) continue;
    if (!/^[A-Za-z0-9 .,!?''\u2019"&\u20AC%\u2026-]+$/.test(value)) continue;
    if (!/[A-Z]|[!?\u2026]/.test(value)) continue;
    issues.push({ file, line: lineOf(source, match.index), text: value, kind: "default" });
  }
}

function lineOf(source, index) {
  return source.slice(0, index).split("\n").length;
}

if (issues.length === 0) {
  console.log("i18n:check ok - no hardcoded UI strings found");
  process.exit(0);
}

const byFile = new Map();
for (const issue of issues) {
  const rel = issue.file.replace(join(SRC, ""), "");
  if (!byFile.has(rel)) byFile.set(rel, []);
  byFile.get(rel).push(`${issue.line}: [${issue.kind}] ${issue.text}`);
}

console.error("i18n:check FAILED - hardcoded UI strings found:");
for (const [file, hits] of byFile) {
  console.error(`  ${file}`);
  for (const hit of hits) console.error(`    ${hit}`);
}
process.exit(1);
