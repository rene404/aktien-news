import { describe, expect, it } from "vitest";
import { feedParams, feedQuery, trendingParams } from "./feed";

const now = new Date("2026-10-28T12:00:00Z");

describe("feedQuery", () => {
  it("spells out every filter so the feed route needs no page defaults", () => {
    expect(feedQuery({ section: "ATX", source: "", range: "24h", cursor: "c1" })).toBe("section=ATX&range=24h");
    expect(feedQuery({ section: "", source: "fn_atx", range: "all", cursor: "" }, "VOE")).toBe("source=fn_atx&range=all&ticker=VOE");
  });
});

describe("feedParams", () => {
  it("round-trips a section query plus cursor into API params", () => {
    const sp = new URLSearchParams(feedQuery({ section: "ATX", source: "", range: "24h", cursor: "" }) + "&cursor=c2");
    expect(feedParams(sp, now)).toEqual({
      index: "ATX",
      kind: "adhoc,press_release,news_api,ir",
      from: "2026-10-27T12:00:00.000Z",
      cursor: "c2",
      limit: 25,
    });
  });

  it("a ticker means the stock page: all kinds, all time by default", () => {
    expect(feedParams(new URLSearchParams("ticker=VOE&section=ATX"), now)).toEqual({ ticker: "VOE", limit: 25 });
  });

  it("rejects a malformed ticker", () => {
    expect(feedParams(new URLSearchParams("ticker=%27%3B--"), now)).toBeNull();
  });
});

describe("trendingParams", () => {
  const now = new Date("2026-11-04T12:34:56.789Z");

  it("counts the last 24 hours of the section, to the minute", () => {
    expect(trendingParams("ATX", now)).toEqual({
      index: "ATX",
      kind: "adhoc,press_release,news_api,ir",
      from: "2026-11-03T12:34:00.000Z",
      limit: 10,
    });
  });

  it("has no index for sections without one", () => {
    expect(trendingParams("filings", now)).toEqual({ kind: "filing", from: "2026-11-03T12:34:00.000Z", limit: 10 });
    expect(trendingParams("", now).index).toBeUndefined();
  });
});
