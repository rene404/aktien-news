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
