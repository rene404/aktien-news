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
] as const;

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
