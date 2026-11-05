import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getNews, type NewsItem } from "@/lib/api";
import NewsFeed from "./NewsFeed";

vi.mock("@/lib/api", () => ({ getNews: vi.fn() }));

const item: NewsItem = {
  id: 1,
  title: "Erste Group erhöht Dividende",
  url: "https://example.com/1",
  source: { key: "fn_atx", name: "FinanzNachrichten ATX", kind: "news_api" },
  published_at: "2026-10-28T10:00:00Z",
  tickers: ["EBS", "OMV"],
};

beforeEach(() => vi.mocked(getNews).mockResolvedValue({ items: [item], next_cursor: "c2" }));

describe("NewsFeed", () => {
  it("on the start page asks for the section's kinds and links the next page with the filters", async () => {
    render(await NewsFeed({ filters: { section: "ATX", source: "", range: "24h", cursor: "" }, path: "/" }));
    expect(getNews).toHaveBeenCalledWith(expect.objectContaining({ index: "ATX", kind: "adhoc,press_release,news_api,ir" }));
    expect(screen.getByRole("link", { name: /ältere meldungen/i })).toHaveAttribute("href", "/?section=ATX&range=24h&cursor=c2");
  });

  it("on a stock page asks for that ticker only, keeps its default range out of the URL and hides its chip", async () => {
    render(await NewsFeed({ filters: { section: "", source: "", range: "all", cursor: "" }, path: "/aktie/EBS", ticker: "EBS" }));
    const params = vi.mocked(getNews).mock.calls.at(-1)![0];
    expect(params).toMatchObject({ ticker: "EBS" });
    expect(params.kind).toBeUndefined();
    expect(screen.getByRole("link", { name: /ältere meldungen/i })).toHaveAttribute("href", "/aktie/EBS?cursor=c2");
    expect(screen.queryByRole("link", { name: "EBS" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "OMV" })).toBeInTheDocument();
  });
});
