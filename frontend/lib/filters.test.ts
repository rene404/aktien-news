import { describe, expect, it } from "vitest";
import type { Source, SourceKind } from "./api";
import { filterHref, parseFilters, sourcesFor, toNewsParams } from "./filters";

const now = new Date("2026-10-28T12:00:00Z");
const NEWS_KINDS = "adhoc,press_release,news_api,ir";

describe("parseFilters", () => {
  it("defaults", () => {
    expect(parseFilters({})).toEqual({ section: "", source: "", range: "7d", cursor: "" });
  });

  it("takes the default range from the caller", () => {
    expect(parseFilters({}, "all").range).toBe("all");
  });

  it("keeps known values and drops unknown ones", () => {
    expect(parseFilters({ section: "ATX", source: "fn_adhoc", range: "24h", cursor: "abc" })).toEqual({
      section: "ATX",
      source: "fn_adhoc",
      range: "24h",
      cursor: "abc",
    });
    expect(parseFilters({ section: "<script>", source: "a b", range: "1y" })).toEqual({
      section: "",
      source: "",
      range: "7d",
      cursor: "",
    });
  });

  it("reads the old index parameter as a section", () => {
    expect(parseFilters({ index: "DAX" }).section).toBe("DAX");
  });

  it("uses the first value of repeated parameters", () => {
    expect(parseFilters({ section: ["DAX", "ATX"] }).section).toBe("DAX");
  });
});

describe("toNewsParams", () => {
  it("an index section keeps filings out and turns the range into a from timestamp", () => {
    expect(toNewsParams({ section: "DAX", source: "", range: "24h", cursor: "" }, now)).toEqual({
      index: "DAX",
      kind: NEWS_KINDS,
      from: "2026-10-27T12:00:00.000Z",
      limit: 25,
    });
  });

  it("the default section shows everything but filings", () => {
    expect(toNewsParams({ section: "", source: "", range: "all", cursor: "" }, now)).toEqual({ kind: NEWS_KINDS, limit: 25 });
  });

  it("kind sections filter by source kind", () => {
    expect(toNewsParams({ section: "adhoc", source: "", range: "all", cursor: "" }, now).kind).toBe("adhoc");
    expect(toNewsParams({ section: "filings", source: "", range: "all", cursor: "" }, now).kind).toBe("filing");
  });

  it("an explicit source wins over the kind filter and passes the cursor", () => {
    expect(toNewsParams({ section: "", source: "sec_8k", range: "all", cursor: "c1" }, now)).toEqual({
      source: "sec_8k",
      cursor: "c1",
      limit: 25,
    });
  });
});

describe("filterHref", () => {
  it("drops defaults and the cursor unless asked", () => {
    expect(filterHref("/", { section: "", source: "", range: "7d", cursor: "abc" })).toBe("/");
    expect(filterHref("/", { section: "ATX", source: "", range: "24h", cursor: "abc" })).toBe("/?section=ATX&range=24h");
  });

  it("keeps the cursor for the next page", () => {
    expect(filterHref("/aktie/EBS", { section: "", source: "fn_atx", range: "all", cursor: "c2" }, { cursor: true, defaultRange: "all" })).toBe(
      "/aktie/EBS?source=fn_atx&cursor=c2",
    );
  });
});

describe("sourcesFor", () => {
  const src = (key: string, market: string, kind: SourceKind): Source => ({ key, name: key, market, kind });
  const all = [
    src("fn_adhoc", "MULTI", "adhoc"),
    src("fn_atx", "AT", "news_api"),
    src("fn_dax", "DE", "news_api"),
    src("prn_all", "US", "press_release"),
    src("sec_8k", "US", "filing"),
  ];
  const keys = (s: Source[]) => s.map((x) => x.key);

  it.each([
    ["", ["fn_adhoc", "fn_atx", "fn_dax", "prn_all"]],
    ["ATX", ["fn_adhoc", "fn_atx"]],
    ["DAX", ["fn_adhoc", "fn_dax"]],
    ["SP500", ["fn_adhoc", "prn_all"]],
    ["adhoc", ["fn_adhoc"]],
    ["filings", ["sec_8k"]],
  ])("section %j offers %j", (section, want) => {
    expect(keys(sourcesFor(all, { section }))).toEqual(want);
  });

  it("limits a stock page to the market of its exchange", () => {
    expect(keys(sourcesFor(all, { mic: "XWBO" }))).toEqual(["fn_adhoc", "fn_atx"]);
    expect(keys(sourcesFor(all, { mic: "XNAS" }))).toEqual(["fn_adhoc", "prn_all", "sec_8k"]);
  });

  it("offers every source for an unknown exchange", () => {
    expect(sourcesFor(all, { mic: "XLON" })).toHaveLength(all.length);
  });

  it("keeps a source picked by URL even if it does not fit", () => {
    expect(keys(sourcesFor(all, { section: "ATX", selected: "sec_8k" }))).toEqual(["fn_adhoc", "fn_atx", "sec_8k"]);
  });
});

describe("adhoc only (stock page)", () => {
  it("is read from ?adhoc=1 and absent otherwise", () => {
    expect(parseFilters({ adhoc: "1" }).adhoc).toBe(true);
    expect(parseFilters({ adhoc: "yes" })).not.toHaveProperty("adhoc");
    expect(parseFilters({})).not.toHaveProperty("adhoc");
  });

  it("is in the URL only when on", () => {
    const f = { section: "", source: "", range: "all", cursor: "" };
    expect(filterHref("/aktie/VOE", { ...f, adhoc: true }, { defaultRange: "all" })).toBe("/aktie/VOE?adhoc=1");
    expect(filterHref("/aktie/VOE", { ...f, adhoc: false }, { defaultRange: "all" })).toBe("/aktie/VOE");
  });
});
