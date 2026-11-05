import { act, fireEvent, render, screen } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import ShareButton from "./ShareButton";

const title = "VOEST hebt Prognose";
const url = "https://example.com/1";

function setNavigator(props: Record<string, unknown>) {
  for (const [k, v] of Object.entries(props)) Object.defineProperty(navigator, k, { value: v, configurable: true });
}

describe("ShareButton", () => {
  afterEach(() => {
    setNavigator({ share: undefined, clipboard: undefined });
    vi.useRealTimers();
  });

  it("is not rendered visibly without JavaScript", () => {
    expect(renderToString(<ShareButton title={title} url={url} />)).toContain("hidden");
  });

  it("stays hidden where the browser can neither share nor copy (plain http)", () => {
    setNavigator({ share: undefined, clipboard: undefined });
    render(<ShareButton title={title} url={url} />);
    expect(screen.getByRole("button", { hidden: true })).not.toBeVisible();
  });

  it("names the news it shares", () => {
    setNavigator({ clipboard: { writeText: vi.fn() } });
    render(<ShareButton title={title} url={url} />);
    expect(screen.getByRole("button", { name: `Teilen: ${title}` })).toBeVisible();
  });

  it("uses the share sheet of the phone with title and the original link", async () => {
    const share = vi.fn().mockResolvedValue(undefined);
    setNavigator({ share });
    render(<ShareButton title={title} url={url} />);
    await act(async () => void fireEvent.click(screen.getByRole("button")));
    expect(share).toHaveBeenCalledWith({ title, url });
  });

  it("stays quiet when the reader cancels the share sheet", async () => {
    setNavigator({ share: vi.fn().mockRejectedValue(new DOMException("cancelled", "AbortError")) });
    render(<ShareButton title={title} url={url} />);
    await act(async () => void fireEvent.click(screen.getByRole("button")));
    expect(screen.queryByRole("status")).toHaveTextContent("");
  });

  it("copies the link where there is no share sheet, and says so for a moment", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const writeText = vi.fn().mockResolvedValue(undefined);
    setNavigator({ share: undefined, clipboard: { writeText } });
    render(<ShareButton title={title} url={url} />);
    await act(async () => void fireEvent.click(screen.getByRole("button")));
    expect(writeText).toHaveBeenCalledWith(url);
    expect(screen.getByRole("status")).toHaveTextContent("Link kopiert");
    await act(async () => void vi.advanceTimersByTime(2500));
    expect(screen.getByRole("status")).toHaveTextContent("");
  });
});
