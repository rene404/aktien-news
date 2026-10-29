import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { NewsItem } from "@/lib/api";
import NewsList, { KIND_LABELS } from "./NewsList";

const now = new Date("2026-10-28T12:00:00Z");
const NEW_WINDOW = "(öffnet in neuem Fenster)";

const items: NewsItem[] = [
  {
    id: 1,
    title: "EQS-Adhoc: Lenzing AG: Prognose angehoben",
    url: "https://www.finanznachrichten.de/nachrichten-2026-10/1.htm",
    source: { key: "fn_adhoc", name: "FinanzNachrichten Ad-hoc", kind: "adhoc" },
    published_at: "2026-10-28T09:30:00Z",
    tickers: ["LNZ"],
  },
  {
    id: 2,
    title: "Siemens und Siemens Energy im Fokus",
    url: "https://example.com/2",
    source: { key: "fn_dax", name: "FinanzNachrichten DAX", kind: "news_api" },
    published_at: "2026-10-28T08:00:00Z",
    tickers: ["ENR", "SIE"],
  },
  {
    id: 3,
    title: "8-K: APPLE INC – Item 2.02 Results of Operations and Financial Condition",
    url: "https://www.sec.gov/3",
    source: { key: "sec_8k", name: "SEC 8-K", kind: "filing" },
    published_at: "2026-10-26T20:30:00Z",
    tickers: ["AAPL"],
  },
];

describe("NewsList", () => {
  it("links each title to the original source in a new window and says so", () => {
    render(<NewsList items={items} now={now} />);
    const link = screen.getByRole("link", { name: `${items[0].title} ${NEW_WINDOW}` });
    expect(link).toHaveAttribute("href", items[0].url);
    // FinanzNachrichten terms: links open in a new window, no frames.
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
  });

  it("groups by Vienna day", () => {
    render(<NewsList items={items} now={now} />);
    const days = screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);
    expect(days).toEqual(["Heute", "Montag, 26.10.2026"]);
    const older = screen.getByRole("region", { name: "Montag, 26.10.2026" });
    expect(within(older).getAllByRole("listitem")).toHaveLength(1);
  });

  it("shows source, local clock time and ticker chips that lead to the stock page", () => {
    render(<NewsList items={items} now={now} />);
    const second = screen.getAllByRole("listitem")[1];
    expect(within(second).getByText("FinanzNachrichten DAX")).toBeInTheDocument();
    expect(within(second).getByRole("link", { name: "ENR" })).toHaveAttribute("href", "/aktie/ENR");
    expect(within(second).getByRole("link", { name: "SIE" })).toHaveAttribute("href", "/aktie/SIE");

    const time = within(screen.getAllByRole("listitem")[0]).getByText("10:30"); // 09:30Z = 10:30 Vienna (CET)
    expect(time.tagName).toBe("TIME");
    expect(time).toHaveAttribute("dateTime", "2026-10-28T09:30:00Z");
  });

  it("badges ad-hoc releases and filings, not plain news", () => {
    render(<NewsList items={items} now={now} />);
    const [adhoc, news, filing] = screen.getAllByRole("listitem");
    expect(within(adhoc).getByText("Ad-hoc")).toHaveClass("badge");
    expect(within(filing).getByText("Filing")).toHaveClass("badge");
    expect(news.querySelector(".badge")).toBeNull();
  });

  it("can hide the chip of the stock the page is about", () => {
    render(<NewsList items={items} now={now} hideTicker="LNZ" />);
    expect(screen.queryByRole("link", { name: "LNZ" })).not.toBeInTheDocument();
  });

  it("renders only what the API sends: no text beyond title, source, time, kind and tickers", () => {
    render(<NewsList items={items} now={now} />);
    screen.getAllByRole("listitem").forEach((li, i) => {
      let rest = li.textContent ?? "";
      const it = items[i];
      for (const s of [it.title, NEW_WINDOW, it.source.name, KIND_LABELS[it.source.kind] ?? "", ...it.tickers, li.querySelector("time")?.textContent ?? ""]) {
        rest = rest.replace(s, "");
      }
      expect(rest.replace(/[\s·↗]/g, "")).toBe("");
    });
  });

  it("shows an empty state", () => {
    render(<NewsList items={[]} now={now} />);
    expect(screen.getByText(/keine meldungen/i)).toBeInTheDocument();
  });
});
