import { describe, expect, it } from "vitest";
import type { TickerCount } from "./api";
import { LIST_INDICES, listedIndices, parseStockList, sortTickers, stockListHref } from "./stocklist";

describe("parseStockList", () => {
  it("defaults to the ATX, 7 days, most news first", () => {
    expect(parseStockList({})).toEqual({ index: "ATX", range: "7d", sort: "count" });
  });

  it("lists the European indices in the order of the tabs", () => {
    expect(LIST_INDICES).toEqual([
      { key: "ATX", label: "ATX" },
      { key: "DAX", label: "DAX" },
      { key: "CAC40", label: "CAC 40" },
      { key: "IBEX35", label: "IBEX 35" },
      { key: "AEX", label: "AEX" },
      { key: "OMXS30", label: "OMXS30" },
      { key: "OMXH25", label: "OMXH25" },
      { key: "OMXC25", label: "OMXC25" },
      { key: "SP500", label: "S&P 500" },
    ]);
    expect(parseStockList({ index: "IBEX35" }).index).toBe("IBEX35");
  });

  it("names the indices as a German list", () => {
    expect(listedIndices()).toBe("ATX, DAX, CAC 40, IBEX 35, AEX, OMXS30, OMXH25, OMXC25 und S&P 500");
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
