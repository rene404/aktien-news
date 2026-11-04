import { describe, expect, it } from "vitest";
import type { NewsItem } from "./api";
import { mergeNews, unseen } from "./news";

const at = (id: number, published_at: string): NewsItem => ({
  id,
  title: `M${id}`,
  url: `https://example.com/${id}`,
  source: { key: "fn_atx", name: "FN", kind: "news_api" },
  published_at,
  tickers: [],
});

const a = at(1, "2026-11-04T10:00:00Z");
const b = at(2, "2026-11-04T11:00:00Z");
const c = at(3, "2026-11-04T12:00:00Z");

describe("unseen", () => {
  it("keeps only items not in the list", () => {
    expect(unseen([b, a], [c, b, a]).map((i) => i.id)).toEqual([3]);
    expect(unseen([b, a], [b, a])).toEqual([]);
  });
});

describe("mergeNews", () => {
  it("puts new items in order, newest first, without duplicates", () => {
    expect(mergeNews([b, a], [c, b]).map((i) => i.id)).toEqual([3, 2, 1]);
  });

  it("files a late item that is older than the top by its time", () => {
    const late = at(9, "2026-11-04T10:30:00Z");
    expect(mergeNews([c, b, a], [late]).map((i) => i.id)).toEqual([3, 2, 9, 1]);
  });

  it("orders equal times by id, like the API", () => {
    const same = at(5, "2026-11-04T11:00:00Z");
    expect(mergeNews([b, a], [same]).map((i) => i.id)).toEqual([5, 2, 1]);
  });
});
