import { afterEach, describe, expect, it, vi } from "vitest";
import { getInstrument, getNews, newsUrl } from "./api";

function mockFetch(status: number, body: unknown) {
  const fn = vi.fn<(input: string) => Promise<Response>>(async () =>
    new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }),
  );
  vi.stubGlobal("fetch", fn);
  return fn;
}

afterEach(() => vi.unstubAllGlobals());

describe("newsUrl", () => {
  it("only includes set parameters", () => {
    expect(newsUrl("http://api:8080", {})).toBe("http://api:8080/api/v1/news");
    expect(newsUrl("http://api:8080", { ticker: "EBS", index: "", cursor: "abc", limit: 25 })).toBe(
      "http://api:8080/api/v1/news?ticker=EBS&cursor=abc&limit=25",
    );
  });

  it("encodes values", () => {
    expect(newsUrl("http://api", { from: "2026-10-28T10:00:00+01:00" })).toBe(
      "http://api/api/v1/news?from=2026-10-28T10%3A00%3A00%2B01%3A00",
    );
  });
});

describe("getNews", () => {
  it("returns the page", async () => {
    const page = { items: [], next_cursor: null };
    const fn = mockFetch(200, page);
    await expect(getNews({ ticker: "EBS" })).resolves.toEqual(page);
    expect(fn).toHaveBeenCalledOnce();
  });

  it("throws on errors so the page shows the error boundary", async () => {
    mockFetch(500, { error: "internal error" });
    await expect(getNews({})).rejects.toThrow(/500/);
  });
});

describe("getInstrument", () => {
  it("returns null for unknown tickers", async () => {
    mockFetch(404, { error: "not found" });
    await expect(getInstrument("NOPE")).resolves.toBeNull();
  });

  it("returns null for tickers the API rejects as invalid", async () => {
    mockFetch(400, { error: "invalid ticker" });
    await expect(getInstrument("a b")).resolves.toBeNull();
  });

  it("encodes the ticker in the path", async () => {
    const fn = mockFetch(200, { ticker: "BRK.B", mic: "XNYS", name: "Berkshire", country: "US", indices: [] });
    await getInstrument("BRK.B");
    expect(String(fn.mock.calls[0][0])).toMatch(/\/api\/v1\/instruments\/BRK\.B$/);
  });
});
