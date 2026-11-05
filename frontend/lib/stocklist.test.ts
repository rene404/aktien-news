import { describe, expect, it } from "vitest";
import type { TickerCount } from "./api";
import { parseStockList, sortTickers, stockListHref } from "./stocklist";

describe("parseStockList", () => {
  it("defaults to the ATX, 7 days, most news first", () => {
    expect(parseStockList({})).toEqual({ index: "ATX", range: "7d", sort: "count" });
  });

  it("keeps known values and drops unknown ones", () => {
    expect(parseStockList({ index: "SP500", range: "24h", sort: "name" })).toEqual({ index: "SP500", range: "24h", sort: "name" });
    expect(parseStockList({ index: "NIKKEI", range: "30d", sort: "x" })).toEqual({ index: "ATX", range: "7d", sort: "count" });
  });
});

describe("stockListHref", () => {
  it("leaves defaults out of the URL", () => {
    expect(stockListHref({ index: "ATX", range: "7d", sort: "count" })).toBe("/aktien");
    expect(stockListHref({ index: "DAX", range: "24h", sort: "name" })).toBe("/aktien?index=DAX&range=24h&sort=name");
  });
});

describe("sortTickers", () => {
  const items: TickerCount[] = [
    { ticker: "POST", name: "Österreichische Post", count: 2 },
    { ticker: "ANDR", name: "Andritz", count: 0 },
    { ticker: "EBS", name: "Erste Group", count: 2 },
    { ticker: "OMV", name: "OMV", count: 5 },
  ];

  it("by count, then by name", () => {
    expect(sortTickers(items, "count").map((i) => i.ticker)).toEqual(["OMV", "EBS", "POST", "ANDR"]);
  });

  it("by name, German order (Ö after O)", () => {
    expect(sortTickers(items, "name").map((i) => i.ticker)).toEqual(["ANDR", "EBS", "OMV", "POST"]);
  });

  it("does not change its input", () => {
    const copy = [...items];
    sortTickers(items, "count");
    expect(items).toEqual(copy);
  });
});
