// @vitest-environment node
import { existsSync, readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { ICON_PUBLISHERS, sourceIcon } from "./sources";

describe("sourceIcon", () => {
  it.each([
    ["fn_adhoc", "/sources/fn.png"],
    ["fn_dax", "/sources/fn.png"],
    ["ots_finanzen", "/sources/ots.png"],
    ["oekb_filings", "/sources/oekb.png"],
    ["prn_all", "/sources/prn.png"],
    ["sec_8k", "/sources/sec.png"],
    ["telekom_ir", "/sources/telekom.png"],
    ["voestalpine_ir", "/sources/voestalpine.png"],
    ["nasdaq_nordic_adhoc", "/sources/nasdaq.png"],
    ["fca_nsm", "/sources/fca.png"],
  ])("maps %s to the icon of its publisher", (key, src) => {
    expect(sourceIcon(key)).toBe(src);
  });

  it("has no icon for an unknown publisher", () => {
    expect(sourceIcon("borsa_news")).toBeNull();
    expect(sourceIcon("")).toBeNull();
  });

  it.each(ICON_PUBLISHERS)("ships public/sources/%s.png", (publisher) => {
    expect(existsSync(new URL(`../public/sources/${publisher}.png`, import.meta.url))).toBe(true);
  });
});

// Regression: output "standalone" leaves public/ out of the bundle; without the
// COPY in the Dockerfile every icon was a 404 in the running container.
describe("Docker image", () => {
  it("copies public/ next to the standalone server", () => {
    const dockerfile = readFileSync(new URL("../Dockerfile", import.meta.url), "utf8");
    expect(dockerfile).toMatch(/^COPY --from=build \/app\/public \.\/public$/m);
  });
});
