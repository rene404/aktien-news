import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { NewsItem, NewsPage } from "@/lib/api";
import InfiniteNews from "./InfiniteNews";

const now = "2026-10-28T12:00:00Z";

function item(id: number, published_at = "2026-10-28T10:00:00Z"): NewsItem {
  return {
    id,
    title: `Meldung ${id}`,
    url: `https://example.com/${id}`,
    source: { key: "fn_atx", name: "FinanzNachrichten ATX", kind: "news_api" },
    published_at,
    tickers: [],
  };
}

// Captures the observer so a test can scroll the sentinel into view.
let intersect: () => void;
class FakeObserver {
  constructor(private cb: IntersectionObserverCallback) {}
  observe() {
    intersect = () => this.cb([{ isIntersecting: true } as IntersectionObserverEntry], this as unknown as IntersectionObserver);
  }
  disconnect() {}
}

function mockFetch(...pages: (NewsPage | number)[]) {
  const fn = vi.fn<(input: string) => Promise<Response>>(async () => {
    const next = pages.shift()!;
    return typeof next === "number"
      ? new Response("{}", { status: next })
      : new Response(JSON.stringify(next), { status: 200, headers: { "Content-Type": "application/json" } });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

beforeEach(() => vi.stubGlobal("IntersectionObserver", FakeObserver));
afterEach(() => vi.unstubAllGlobals());

function renderList(nextCursor: string | null = "c1") {
  return render(
    <InfiniteNews items={[item(1)]} nextCursor={nextCursor} query="section=ATX&range=7d" moreHref="/?section=ATX&cursor=c1" now={now} />,
  );
}

describe("InfiniteNews", () => {
  it("keeps a plain link to the next page for browsers without JavaScript", () => {
    renderList();
    expect(screen.getByRole("link", { name: /ältere meldungen/i })).toHaveAttribute("href", "/?section=ATX&cursor=c1");
  });

  it("loads and appends the next page when the end of the list scrolls into view", async () => {
    const fn = mockFetch({ items: [item(2, "2026-10-27T10:00:00Z")], next_cursor: "c2" });
    renderList();
    await act(async () => intersect());
    expect(fn).toHaveBeenCalledWith("/feed?section=ATX&range=7d&cursor=c1");
    expect(screen.getByText("Meldung 1")).toBeInTheDocument();
    expect(screen.getByText("Meldung 2")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Gestern" })).toBeInTheDocument();
  });

  it("asks for the following cursor next and stops on the last page", async () => {
    const fn = mockFetch({ items: [item(2)], next_cursor: "c2" }, { items: [item(3)], next_cursor: null });
    renderList();
    await act(async () => intersect());
    await act(async () => intersect());
    expect(fn).toHaveBeenLastCalledWith("/feed?section=ATX&range=7d&cursor=c2");
    expect(screen.getByText("Meldung 3")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /ältere meldungen/i })).not.toBeInTheDocument();
  });

  it("does not show an item twice when pages overlap", async () => {
    mockFetch({ items: [item(1), item(2)], next_cursor: null });
    renderList();
    await act(async () => intersect());
    expect(screen.getAllByText("Meldung 1")).toHaveLength(1);
  });

  it("offers a retry when loading fails", async () => {
    mockFetch(502, { items: [item(2)], next_cursor: null });
    renderList();
    await act(async () => intersect());
    const retry = screen.getByRole("link", { name: /erneut versuchen/i });
    await act(async () => fireEvent.click(retry));
    expect(screen.getByText("Meldung 2")).toBeInTheDocument();
  });

  it("has nothing to load without a cursor", () => {
    const fn = mockFetch();
    renderList(null);
    expect(screen.queryByRole("link", { name: /ältere meldungen/i })).not.toBeInTheDocument();
    expect(fn).not.toHaveBeenCalled();
  });
});
