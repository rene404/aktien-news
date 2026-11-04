import { act, fireEvent, render } from "@testing-library/react";
import { useRef } from "react";
import { describe, expect, it } from "vitest";
import { useScrollRow } from "./useScrollRow";

function Row({ active }: { active?: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  useScrollRow(ref);
  return (
    <div ref={ref} data-testid="row">
      <a>eins</a>
      <a aria-current={active ? "page" : undefined}>zwei</a>
    </div>
  );
}

// jsdom has no layout: give the row and its active link fixed sizes.
function layout(el: HTMLElement, { scrollWidth, clientWidth, scrollLeft = 0 }: { scrollWidth: number; clientWidth: number; scrollLeft?: number }) {
  Object.defineProperty(el, "scrollWidth", { configurable: true, value: scrollWidth });
  Object.defineProperty(el, "clientWidth", { configurable: true, value: clientWidth });
  el.scrollLeft = scrollLeft;
}

describe("useScrollRow", () => {
  it("marks the edges that have more content", () => {
    const { getByTestId } = render(<Row />);
    const row = getByTestId("row");

    layout(row, { scrollWidth: 300, clientWidth: 300 });
    fireEvent.scroll(row);
    expect(row).not.toHaveAttribute("data-more-start");
    expect(row).not.toHaveAttribute("data-more-end");

    layout(row, { scrollWidth: 600, clientWidth: 300, scrollLeft: 0 });
    fireEvent.scroll(row);
    expect(row).not.toHaveAttribute("data-more-start");
    expect(row).toHaveAttribute("data-more-end");

    layout(row, { scrollWidth: 600, clientWidth: 300, scrollLeft: 150 });
    fireEvent.scroll(row);
    expect(row).toHaveAttribute("data-more-start");
    expect(row).toHaveAttribute("data-more-end");

    layout(row, { scrollWidth: 600, clientWidth: 300, scrollLeft: 300 });
    fireEvent.scroll(row);
    expect(row).toHaveAttribute("data-more-start");
    expect(row).not.toHaveAttribute("data-more-end");
  });

  it("updates on window resize", () => {
    const { getByTestId } = render(<Row />);
    const row = getByTestId("row");
    layout(row, { scrollWidth: 600, clientWidth: 300 });
    act(() => {
      window.dispatchEvent(new Event("resize"));
    });
    expect(row).toHaveAttribute("data-more-end");
  });
});

describe("useScrollRow on mount", () => {
  it("brings a current item outside the view into the middle", () => {
    const { getByTestId } = render(<RowWithLayout />);
    expect(getByTestId("row").scrollLeft).toBe(400 + 50 - 150);
  });
});

// Layout must exist before the effect runs, so it is set in a ref callback.
function RowWithLayout() {
  const ref = useRef<HTMLDivElement>(null);
  useScrollRow(ref);
  return (
    <div
      ref={(el) => {
        ref.current = el;
        if (!el) return;
        layout(el, { scrollWidth: 600, clientWidth: 300 });
        const a = el.querySelector<HTMLElement>("[aria-current]")!;
        Object.defineProperty(a, "offsetLeft", { configurable: true, value: 400 });
        Object.defineProperty(a, "offsetWidth", { configurable: true, value: 100 });
      }}
      data-testid="row"
    >
      <a aria-current="page">weit rechts</a>
    </div>
  );
}
