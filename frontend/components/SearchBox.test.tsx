import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import SearchBox, { SUGGEST_DELAY_MS } from "./SearchBox";

const push = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

const EBS = { ticker: "EBS", mic: "XWBO", name: "Erste Group" };
const EVN = { ticker: "EVN", mic: "XWBO", name: "EVN" };

function reply(items: unknown[]) {
  return new Response(JSON.stringify({ items }), { status: 200, headers: { "Content-Type": "application/json" } });
}

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.useFakeTimers();
  fetchMock = vi.fn(async () => reply([EBS, EVN]));
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  push.mockReset();
});

const input = () => screen.getByRole("combobox", { name: "Aktie suchen" });

async function type(value: string) {
  fireEvent.change(input(), { target: { value } });
  await act(async () => {
    await vi.advanceTimersByTimeAsync(SUGGEST_DELAY_MS);
  });
}

describe("SearchBox", () => {
  it("stays a plain GET form to /suche", () => {
    render(<SearchBox />);
    expect(screen.getByRole("search")).toHaveAttribute("action", "/suche");
    expect(input()).toHaveAttribute("name", "q");
  });

  it("asks once the typing pauses, from two characters on", async () => {
    render(<SearchBox />);
    await type("e");
    expect(fetchMock).not.toHaveBeenCalled();
    fireEvent.change(input(), { target: { value: "er" } });
    fireEvent.change(input(), { target: { value: "ers" } });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(SUGGEST_DELAY_MS);
    });
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(String(fetchMock.mock.calls[0][0])).toBe("/suggest?q=ers");
    expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual(["Erste GroupEBS", "EVNEVN"]);
    expect(input()).toHaveAttribute("aria-expanded", "true");
  });

  it("arrow keys pick, Enter opens the stock", async () => {
    render(<SearchBox />);
    await type("er");
    fireEvent.keyDown(input(), { key: "ArrowDown" });
    fireEvent.keyDown(input(), { key: "ArrowDown" });
    fireEvent.keyDown(input(), { key: "ArrowUp" });
    const first = screen.getAllByRole("option")[0];
    expect(first).toHaveAttribute("aria-selected", "true");
    expect(input()).toHaveAttribute("aria-activedescendant", first.id);
    fireEvent.keyDown(input(), { key: "Enter" });
    expect(push).toHaveBeenCalledWith("/aktie/EBS");
  });

  it("Enter without a pick submits the form as before", async () => {
    render(<SearchBox />);
    await type("er");
    const notPrevented = fireEvent.keyDown(input(), { key: "Enter" });
    expect(notPrevented).toBe(true);
    expect(push).not.toHaveBeenCalled();
  });

  it("a click opens the stock", async () => {
    render(<SearchBox />);
    await type("er");
    fireEvent.click(screen.getByRole("option", { name: /EVN/ }));
    expect(push).toHaveBeenCalledWith("/aktie/EVN");
  });

  it("Escape and leaving the field close the list", async () => {
    render(<SearchBox />);
    await type("er");
    fireEvent.keyDown(input(), { key: "Escape" });
    expect(input()).toHaveAttribute("aria-expanded", "false");
    fireEvent.focus(input());
    expect(input()).toHaveAttribute("aria-expanded", "true");
    fireEvent.blur(input());
    expect(input()).toHaveAttribute("aria-expanded", "false");
  });

  it("an older answer never replaces a newer one", async () => {
    let releaseOld: (r: Response) => void = () => {};
    fetchMock.mockImplementationOnce(() => new Promise<Response>((r) => (releaseOld = r)));
    fetchMock.mockImplementationOnce(async () => reply([EVN]));
    render(<SearchBox />);
    await type("er"); // pending
    await type("ev"); // answered
    await act(async () => {
      releaseOld(reply([EBS]));
      await vi.runAllTimersAsync();
    });
    expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual(["EVNEVN"]);
  });

  it("closes again below two characters", async () => {
    render(<SearchBox />);
    await type("er");
    fireEvent.change(input(), { target: { value: "e" } });
    expect(input()).toHaveAttribute("aria-expanded", "false");
  });
});
