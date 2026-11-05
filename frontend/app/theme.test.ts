// @vitest-environment node
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// The look is changed in theme.css alone: globals.css must not define colours or fonts of its own.
describe("theme", () => {
  const globals = readFileSync(new URL("./globals.css", import.meta.url), "utf8").replace(/\/\*[\s\S]*?\*\//g, "");

  it("globals.css uses no literal colours", () => {
    expect(globals.match(/#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(/gi)).toBeNull();
  });

  it("globals.css names no font family", () => {
    const values = [...globals.matchAll(/\bfont(?:-family)?:\s*([^;]+);/g)].map((m) => m[1].trim());
    expect(values.filter((v) => v !== "inherit" && !v.startsWith("var("))).toEqual([]);
  });

  it("every variable globals.css uses is defined in theme.css", () => {
    const theme = readFileSync(new URL("./theme.css", import.meta.url), "utf8");
    const used = new Set([...globals.matchAll(/var\((--[\w-]+)\)/g)].map((m) => m[1]));
    const missing = [...used].filter((v) => !theme.includes(`${v}:`));
    expect(missing).toEqual([]);
  });
});

// Calmer typography (docs/plan-web-inspiration.md, task 1): figures in a monospaced
// face, hairline separators, small uppercase labels.
describe("typography", () => {
  const theme = readFileSync(new URL("./theme.css", import.meta.url), "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  const globals = readFileSync(new URL("./globals.css", import.meta.url), "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  const rule = (selector: string) => {
    const at = globals.indexOf(`${selector} {`);
    if (at < 0) throw new Error(`no rule for ${selector}`);
    return globals.slice(at, globals.indexOf("}", at));
  };

  it("defines the new tokens", () => {
    for (const name of ["--font-mono", "--hairline", "--micro-size", "--micro-tracking"]) {
      expect(theme).toContain(`${name}:`);
    }
  });

  it("loads no font or file at runtime (no third-party requests)", () => {
    expect(theme + globals).not.toMatch(/url\(|@import|@font-face/);
  });

  it.each(["ol.news time", ".badge", ".tchip-count", ".chip", ".tchip-ticker", ".suggest-ticker"])(
    "%s sets figures in the mono face",
    (selector) => {
      expect(rule(selector)).toContain("var(--font-mono)");
    },
  );

  it.each([".skeleton-row", "section.day h2", "ul.results li"])("%s is separated by a hairline", (selector) => {
    expect(rule(selector)).toContain("var(--hairline)");
  });

  it.each(["section.day h2", ".badge", ".trending-label"])("%s is a micro label", (selector) => {
    const r = rule(selector);
    expect(r).toContain("var(--micro-size)");
    expect(r).toContain("var(--micro-tracking)");
    expect(r).toContain("uppercase");
  });
});

// Text/background pairs that must stay readable (WCAG AA, 4.5:1) in every colour scheme theme.css defines.
const PAIRS = [
  ["--text", "--bg"],
  ["--muted", "--bg"],
  ["--link", "--bg"],
  ["--on-brand", "--brand"],
  ["--on-brand", "--button-bg"],
  ["--chip-text", "--chip-bg"],
  ["--header-text", "--header-bg"],
  ["--header-muted", "--header-bg"],
  ["--adhoc", "--adhoc-bg"],
  ["--muted", "--count-zero-bg"],
  ["--text", "--count-zero-bg"], // hovered search suggestion
  ["--link", "--count-zero-bg"],
  ["--link-hover", "--bg"],
  ["--on-brand", "--button-hover"],
] as const;

// Dark mode follows the system setting (docs/plan-web-inspiration.md, task 5).
describe("dark mode", () => {
  const theme = readFileSync(new URL("./theme.css", import.meta.url), "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  const split = theme.indexOf("@media (prefers-color-scheme: dark)");

  it("has a dark scheme and tells the browser both schemes are supported", () => {
    expect(split).toBeGreaterThan(0);
    expect(theme.slice(0, split)).toMatch(/color-scheme:\s*light dark;/);
  });

  it("redefines every literal colour of the light scheme, so none is forgotten", () => {
    const light = tokens(theme.slice(0, split));
    const dark = tokens(theme.slice(split));
    const colours = Object.entries(light)
      .filter(([name, v]) => /^(#|rgba?\()/.test(v) && name !== "--mask-solid") // the mask is not a visible colour
      .map(([name]) => name);
    expect(colours.filter((name) => !(name in dark))).toEqual([]);
  });
});

describe("theme contrast", () => {
  const theme = readFileSync(new URL("./theme.css", import.meta.url), "utf8");
  const split = theme.indexOf("@media (prefers-color-scheme: dark)");
  const light = tokens(split < 0 ? theme : theme.slice(0, split));
  const schemes: [string, Record<string, string>][] = [["light", light]];
  if (split >= 0) schemes.push(["dark", { ...light, ...tokens(theme.slice(split)) }]);

  for (const [scheme, t] of schemes) {
    it.each(PAIRS)(`${scheme}: %s on %s`, (fg, bg) => {
      expect(contrast(resolve(t, fg), resolve(t, bg))).toBeGreaterThanOrEqual(4.5);
    });
  }
});

function tokens(css: string): Record<string, string> {
  return Object.fromEntries([...css.matchAll(/(--[\w-]+):\s*([^;]+);/g)].map((m) => [m[1], m[2].trim()]));
}

function resolve(t: Record<string, string>, name: string): string {
  let v = t[name];
  for (let ref = v?.match(/^var\((--[\w-]+)\)$/); ref; ref = v?.match(/^var\((--[\w-]+)\)$/)) v = t[ref[1]];
  if (!v || !/^#[0-9a-f]{6}$/i.test(v)) throw new Error(`${name} must be a #rrggbb colour, got ${v}`);
  return v;
}

function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}
