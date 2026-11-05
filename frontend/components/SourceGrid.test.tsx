import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { NewsItem, Source } from "@/lib/api";
import SourceGrid from "./SourceGrid";

const src = (key: string, name: string): Source => ({ key, name, market: "AT", kind: "news_api", last_ok_at: null });
const item = (id: number, title: string): NewsItem => ({
  id,
  title,
  url: `https://example.com/${id}`,
  source: { key: "fn_atx", name: "FinanzNachrichten ATX", kind: "news_api" },
  published_at: "2026-11-05T10:00:00Z",
  tickers: ["VOE"],
});

const tiles = [
  { source: src("fn_atx", "FinanzNachrichten ATX"), items: [item(1, "VOEST hebt Prognose"), item(2, "ATX im Plus")], moreHref: "/?section=ATX&source=fn_atx" },
  { source: src("telekom_ir", "Deutsche Telekom IR"), items: [], moreHref: "/?source=telekom_ir" },
  { source: src("ots_finanzen", "APA-OTS Finanzen"), items: [item(3, "Bilanz Q3")], moreHref: "/?section=ATX&source=ots_finanzen" },
];

describe("SourceGrid", () => {
  it("shows one tile per source with news, named after the source; empty sources are left out", () => {
    render(<SourceGrid tiles={tiles} now="2026-11-05T12:00:00Z" />);
    const names = screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);
    expect(names).toEqual(["FinanzNachrichten ATX", "APA-OTS Finanzen"]);
  });

  it("links each title to the original in a new window, and the tile to all news of the source", () => {
    render(<SourceGrid tiles={tiles} now="2026-11-05T12:00:00Z" />);
    const tile = screen.getByRole("region", { name: "FinanzNachrichten ATX" });
    const link = within(tile).getByRole("link", { name: /VOEST hebt Prognose/ });
    expect(link).toHaveAttribute("href", "https://example.com/1");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
    expect(within(tile).getByRole("link", { name: "Alle Meldungen von FinanzNachrichten ATX" })).toHaveAttribute("href", "/?section=ATX&source=fn_atx");
    expect(within(tile).getAllByRole("listitem")).toHaveLength(2);
  });

  it("renders nothing beyond title, time and source: no ticker chips, no text", () => {
    render(<SourceGrid tiles={tiles} now="2026-11-05T12:00:00Z" />);
    const li = within(screen.getByRole("region", { name: "APA-OTS Finanzen" })).getByRole("listitem");
    const rest = (li.textContent ?? "").replace("Bilanz Q3", "").replace(li.querySelector("time")!.textContent!, "").replace("(öffnet in neuem Fenster)", "");
    expect(rest.replace(/[\s↗]/g, "")).toBe("");
  });

  it("says so when no source has news", () => {
    render(<SourceGrid tiles={[tiles[1]]} now="2026-11-05T12:00:00Z" />);
    expect(screen.getByText(/keine meldungen/i)).toBeInTheDocument();
  });
});
