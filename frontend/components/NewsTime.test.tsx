import { act, render, screen } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import NewsTime from "./NewsTime";

const NOW = new Date("2026-11-05T20:00:00Z"); // 21:00 in Vienna
const twelveMinAgo = "2026-11-05T19:48:00Z"; // 20:48 in Vienna

describe("NewsTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });
  afterEach(() => vi.useRealTimers());

  it("renders the clock time on the server, so the HTML does not depend on when it was cached", () => {
    const html = renderToString(<NewsTime iso={twelveMinAgo} />);
    expect(html).toContain("20:48");
    expect(html).not.toContain("vor");
  });

  it("shows minutes ago in the browser, with the full time as title and datetime", () => {
    const { container } = render(<NewsTime iso={twelveMinAgo} />);
    const time = container.querySelector("time")!;
    // "12 Min" on screen, "vor 12 Min" for screen readers.
    expect(time).toHaveTextContent(/^vor 12 Min$/);
    expect(time.querySelector(".sr-only")).toHaveTextContent("vor");
    expect(time).toHaveAttribute("datetime", twelveMinAgo);
    expect(time).toHaveAttribute("title", "05.11.2026, 20:48");
  });

  it("counts up while the page stays open and turns into the clock time after an hour", () => {
    const { container } = render(<NewsTime iso={twelveMinAgo} />);
    const time = container.querySelector("time")!;
    act(() => vi.advanceTimersByTime(60_000));
    expect(time).toHaveTextContent(/^vor 13 Min$/);
    act(() => vi.advanceTimersByTime(48 * 60_000));
    expect(time).toHaveTextContent(/^20:48$/);
  });

  it("reads 'jetzt' within the first minute", () => {
    render(<NewsTime iso="2026-11-05T19:59:30Z" />);
    expect(screen.getByText("jetzt").tagName).toBe("TIME");
  });
});
