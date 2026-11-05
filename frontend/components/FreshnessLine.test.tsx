import { render } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Source } from "@/lib/api";
import FreshnessLine from "./FreshnessLine";

const NOW = new Date("2026-11-05T20:00:00Z"); // 21:00 in Vienna
const src = (key: string, last: string | null): Source => ({ key, name: key, market: "AT", kind: "news_api", last_ok_at: last });
const sources = [src("fn_atx", "2026-11-05T19:55:30Z"), src("ots_finanzen", "2026-11-05T19:40:00Z"), src("voestalpine_ir", null)];

// The whole line as read, across <p> and <time>.
const line = () => document.querySelector("p.freshness")?.textContent;
const text = (html: string) => html.replace(/<[^>]+>/g, "");

describe("FreshnessLine", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });
  afterEach(() => vi.useRealTimers());

  it("counts the sources and names the newest successful fetch", () => {
    render(<FreshnessLine sources={sources} />);
    expect(line()).toBe("3 Quellen · zuletzt aktualisiert vor 4 Min");
  });

  it("renders the clock time on the server, so cached HTML does not go stale", () => {
    const html = renderToString(<FreshnessLine sources={sources} />);
    expect(text(html)).toBe("3 Quellen · zuletzt aktualisiert um 20:55");
  });

  it("says 'gerade eben' within the first minute and the clock time after an hour", () => {
    const { rerender } = render(<FreshnessLine sources={[src("a", "2026-11-05T19:59:40Z")]} />);
    expect(line()).toBe("1 Quelle · zuletzt aktualisiert gerade eben");
    rerender(<FreshnessLine sources={[src("a", "2026-11-05T18:30:00Z")]} />);
    expect(line()).toBe("1 Quelle · zuletzt aktualisiert um 19:30");
  });

  it("names the date when the newest fetch was not today", () => {
    render(<FreshnessLine sources={[src("a", "2026-11-03T08:00:00Z")]} />);
    expect(line()).toBe("1 Quelle · zuletzt aktualisiert am 03.11.2026, 09:00");
  });

  it("shows only the count before the first fetch, and nothing without sources", () => {
    const { container, rerender } = render(<FreshnessLine sources={[src("a", null), src("b", null)]} />);
    expect(container).toHaveTextContent(/^2 Quellen$/);
    rerender(<FreshnessLine sources={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("carries the exact time for machines and the tooltip", () => {
    render(<FreshnessLine sources={sources} />);
    const time = document.querySelector("p.freshness time")!;
    expect(time).toHaveAttribute("datetime", "2026-11-05T19:55:30Z");
    expect(time).toHaveAttribute("title", "05.11.2026, 20:55");
  });
});
