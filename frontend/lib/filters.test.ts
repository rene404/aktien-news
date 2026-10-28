import { describe, expect, it } from "vitest";
import { parseFilters, toNewsParams } from "./filters";

const now = new Date("2026-10-28T12:00:00Z");

describe("parseFilters", () => {
  it("defaults", () => {
    expect(parseFilters({})).toEqual({ index: "", source: "", range: "7d", cursor: "" });
  });

  it("keeps known values and drops unknown ones", () => {
    expect(parseFilters({ index: "ATX", source: "fn_adhoc", range: "24h", cursor: "abc" })).toEqual({
      index: "ATX",
      source: "fn_adhoc",
      range: "24h",
      cursor: "abc",
    });
    expect(parseFilters({ index: "<script>", source: "a b", range: "1y" })).toEqual({
      index: "",
      source: "",
      range: "7d",
      cursor: "",
    });
  });

  it("uses the first value of repeated parameters", () => {
    expect(parseFilters({ index: ["DAX", "ATX"] }).index).toBe("DAX");
  });
});

describe("toNewsParams", () => {
  it("turns the range into a from timestamp", () => {
    expect(toNewsParams({ index: "DAX", source: "", range: "24h", cursor: "" }, now)).toEqual({
      index: "DAX",
      from: "2026-10-27T12:00:00.000Z",
      limit: 25,
    });
  });

  it("'all' sets no lower bound and passes the cursor", () => {
    expect(toNewsParams({ index: "", source: "fn_atx", range: "all", cursor: "c1" }, now)).toEqual({
      source: "fn_atx",
      cursor: "c1",
      limit: 25,
    });
  });
});
