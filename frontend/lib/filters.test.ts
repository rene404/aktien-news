import { describe, expect, it } from "vitest";
import { filterHref, parseFilters, toNewsParams } from "./filters";

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
