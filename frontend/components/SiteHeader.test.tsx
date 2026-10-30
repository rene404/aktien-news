import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import SiteHeader, { SCROLL_IDLE_MS } from "./SiteHeader";

const HEADER_HEIGHT = 60;

function scrollTo(y: number) {
  Object.defineProperty(window, "scrollY", { value: y, configurable: true, writable: true });
  act(() => {
    fireEvent.scroll(window);
  });
}

function header() {
  return screen.getByRole("banner");
}

beforeEach(() => {
  vi.useFakeTimers();
  // jsdom has no layout; give the header a height for the "near the top" rule.
  vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(HEADER_HEIGHT);
  Object.defineProperty(window, "scrollY", { value: 0, configurable: true, writable: true });
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("SiteHeader", () => {
  it("keeps the search form", () => {
    render(<SiteHeader />);
    expect(screen.getByRole("search")).toHaveAttribute("action", "/suche");
  });

  it("slides away while scrolling down and comes back once scrolling stops", () => {
    render(<SiteHeader />);
    scrollTo(400);
    expect(header()).toHaveAttribute("data-hidden");
    act(() => vi.advanceTimersByTime(SCROLL_IDLE_MS - 1));
    expect(header()).toHaveAttribute("data-hidden");
    act(() => vi.advanceTimersByTime(1));
    expect(header()).not.toHaveAttribute("data-hidden");
  });

  it("keeps hiding while the scrolling goes on", () => {
    render(<SiteHeader />);
    scrollTo(400);
    act(() => vi.advanceTimersByTime(SCROLL_IDLE_MS - 50));
    scrollTo(600);
    act(() => vi.advanceTimersByTime(SCROLL_IDLE_MS - 50));
    expect(header()).toHaveAttribute("data-hidden");
  });

  it("shows at once when scrolling up", () => {
    render(<SiteHeader />);
    scrollTo(800);
    scrollTo(700);
    expect(header()).not.toHaveAttribute("data-hidden");
  });

  it("never hides near the top of the page", () => {
    render(<SiteHeader />);
    scrollTo(HEADER_HEIGHT);
    expect(header()).not.toHaveAttribute("data-hidden");
  });

  it("floats with a shadow once the page is scrolled, not at the very top", () => {
    render(<SiteHeader />);
    scrollTo(400);
    act(() => vi.advanceTimersByTime(SCROLL_IDLE_MS));
    expect(header()).toHaveAttribute("data-floating");
    scrollTo(0);
    expect(header()).not.toHaveAttribute("data-floating");
  });

  it("stays while the search has focus and comes back when it gets focus", () => {
    render(<SiteHeader />);
    scrollTo(400);
    act(() => screen.getByRole("searchbox").focus());
    expect(header()).not.toHaveAttribute("data-hidden");
    scrollTo(600);
    expect(header()).not.toHaveAttribute("data-hidden");
  });
});
