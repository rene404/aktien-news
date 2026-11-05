import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getTopTickers } from "@/lib/api";
import Trending from "./Trending";

vi.mock("@/lib/api", () => ({ getTopTickers: vi.fn() }));
const top = vi.mocked(getTopTickers);
afterEach(() => top.mockReset());

describe("Trending", () => {
  it("shows the most-mentioned stocks of the section in API order", async () => {
    top.mockResolvedValue([
      { ticker: "OMV", name: "OMV", count: 5 },
      { ticker: "EBS", name: "Erste Group", count: 2 },
    ]);
    render(await Trending({ section: "ATX" }));
    expect(top).toHaveBeenCalledWith(expect.objectContaining({ index: "ATX", limit: 10 }));
    const list = screen.getByRole("list", { name: "Heute im Gespräch" });
    expect(Array.from(list.querySelectorAll("a")).map((a) => a.getAttribute("href"))).toEqual(["/aktie/OMV", "/aktie/EBS"]);
  });

  // Regression: in a sideways-scrolling row the last chips were cut off by the faded edge.
  it("wraps instead of scrolling, so every chip is visible", async () => {
    top.mockResolvedValue([{ ticker: "OMV", name: "OMV", count: 5 }]);
    render(await Trending({ section: "" }));
    expect(screen.getByRole("list", { name: "Heute im Gespräch" }).closest(".scroll-row")).toBeNull();
  });

  it("renders nothing without mentions", async () => {
    top.mockResolvedValue([]);
    expect(await Trending({ section: "" })).toBeNull();
  });
});
