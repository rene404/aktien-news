import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { NewsItem } from "@/lib/api";
import NewsList from "./NewsList";

const items: NewsItem[] = [
  {
    id: 1,
    title: "EQS-Adhoc: Lenzing AG: Prognose angehoben",
    url: "https://www.finanznachrichten.de/nachrichten-2026-10/1.htm",
    source: { key: "fn_adhoc", name: "FinanzNachrichten.de" },
    published_at: "2026-10-28T09:30:00Z",
    tickers: ["LNZ"],
  },
  {
    id: 2,
    title: "Siemens und Siemens Energy im Fokus",
    url: "https://example.com/2",
    source: { key: "fn_dax", name: "FinanzNachrichten.de DAX" },
    published_at: "2026-10-28T08:00:00Z",
    tickers: ["ENR", "SIE"],
  },
];

describe("NewsList", () => {
  it("links each title to the original source in a new window", () => {
    render(<NewsList items={items} />);
    const link = screen.getByRole("link", { name: "EQS-Adhoc: Lenzing AG: Prognose angehoben" });
    expect(link).toHaveAttribute("href", items[0].url);
    // FinanzNachrichten terms: links open in a new window, no frames.
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
  });

  it("shows source, local time and ticker chips that lead to the stock page", () => {
    render(<NewsList items={items} />);
    const second = screen.getAllByRole("listitem")[1];
    expect(within(second).getByText("FinanzNachrichten.de DAX")).toBeInTheDocument();
    expect(within(second).getByRole("link", { name: "ENR" })).toHaveAttribute("href", "/aktie/ENR");
    expect(within(second).getByRole("link", { name: "SIE" })).toHaveAttribute("href", "/aktie/SIE");

    const time = within(screen.getAllByRole("listitem")[0]).getByText(/10:30/); // 09:30Z = 10:30 Vienna (CET)
    expect(time.tagName).toBe("TIME");
    expect(time).toHaveAttribute("dateTime", "2026-10-28T09:30:00Z");
  });

  it("can hide the chip of the stock the page is about", () => {
    render(<NewsList items={items} hideTicker="LNZ" />);
    expect(screen.queryByRole("link", { name: "LNZ" })).not.toBeInTheDocument();
  });

  it("renders only what the API sends: no text beyond title, source, time and tickers", () => {
    render(<NewsList items={items} />);
    screen.getAllByRole("listitem").forEach((li, i) => {
      let rest = li.textContent ?? "";
      for (const s of [items[i].title, items[i].source.name, ...items[i].tickers, li.querySelector("time")?.textContent ?? ""]) {
        rest = rest.replace(s, "");
      }
      expect(rest.replace(/[\s·]/g, "")).toBe("");
    });
  });

  it("shows an empty state", () => {
    render(<NewsList items={[]} />);
    expect(screen.getByText(/keine meldungen/i)).toBeInTheDocument();
  });
});
