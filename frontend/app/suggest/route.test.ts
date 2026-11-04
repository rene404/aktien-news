// @vitest-environment node
import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { searchInstruments } from "@/lib/api";
import { GET } from "./route";

vi.mock("@/lib/api", () => ({ searchInstruments: vi.fn() }));
const search = vi.mocked(searchInstruments);
afterEach(() => search.mockReset());

const req = (q: string) => new NextRequest(`http://localhost/suggest?q=${encodeURIComponent(q)}`);

describe("GET /suggest", () => {
  it("returns ticker, name and exchange of the matches, cacheable", async () => {
    search.mockResolvedValue([{ ticker: "EBS", mic: "XWBO", name: "Erste Group", isin: "AT0000652011" }]);
    const res = await GET(req(" erst "));
    expect(search).toHaveBeenCalledWith("erst");
    expect(await res.json()).toEqual({ items: [{ ticker: "EBS", mic: "XWBO", name: "Erste Group" }] });
    expect(res.headers.get("Cache-Control")).toBe("public, max-age=60");
  });

  it("asks nothing for fewer than two characters or more than 64", async () => {
    for (const q of ["", "e", "x".repeat(65)]) {
      expect(await (await GET(req(q))).json()).toEqual({ items: [] });
    }
    expect(search).not.toHaveBeenCalled();
  });

  it("is empty when the API fails", async () => {
    search.mockRejectedValue(new Error("API 500"));
    const res = await GET(req("erst"));
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ items: [] });
  });
});
