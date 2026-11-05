import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import Shortcuts from "./Shortcuts";

function page() {
  return render(
    <>
      <form className="search">
        <input aria-label="Suche" />
      </form>
      <ol className="news">
        {[1, 2, 3].map((n) => (
          <li key={n}>
            <a className="title" href={`https://example.com/${n}`} target="_blank" rel="noopener noreferrer">
              Meldung {n}
            </a>
          </li>
        ))}
      </ol>
      <Shortcuts />
    </>,
  );
}

const press = (key: string, init: KeyboardEventInit = {}) => act(() => void fireEvent.keyDown(document.activeElement ?? document.body, { key, ...init }));
const focused = () => document.activeElement?.textContent;

describe("Shortcuts", () => {
  afterEach(() => vi.restoreAllMocks());

  it("j and k move through the news titles", () => {
    page();
    press("j");
    expect(focused()).toBe("Meldung 1");
    press("j");
    press("j");
    press("j"); // stays on the last one
    expect(focused()).toBe("Meldung 3");
    press("k");
    expect(focused()).toBe("Meldung 2");
    press("k");
    press("k"); // stays on the first one
    expect(focused()).toBe("Meldung 1");
  });

  it("o opens the current title at the source in a new window", () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    page();
    press("o"); // nothing chosen yet: nothing happens
    expect(open).not.toHaveBeenCalled();
    press("j");
    press("j");
    press("o");
    expect(open).toHaveBeenCalledWith("https://example.com/2", "_blank", "noopener,noreferrer");
  });

  it("/ jumps into the search", () => {
    page();
    press("/");
    expect(document.activeElement).toBe(screen.getByRole("textbox", { name: "Suche" }));
  });

  it("does nothing while typing in a field or with a modifier key", () => {
    page();
    screen.getByRole("textbox", { name: "Suche" }).focus();
    press("j");
    expect(document.activeElement).toBe(screen.getByRole("textbox", { name: "Suche" }));
    (document.activeElement as HTMLElement).blur();
    press("j", { metaKey: true });
    press("j", { ctrlKey: true });
    expect(focused()).not.toBe("Meldung 1");
  });

  it("? shows the help, Escape and the button close it", () => {
    page();
    expect(screen.queryByRole("dialog")).toBeNull();
    press("?");
    const help = screen.getByRole("dialog", { name: "Tastenkürzel" });
    expect(help).toHaveTextContent("j");
    expect(help).toHaveTextContent("nächste Meldung");
    press("Escape");
    expect(screen.queryByRole("dialog")).toBeNull();
    press("?");
    fireEvent.click(screen.getByRole("button", { name: "Schließen" }));
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("offers the help as a button, shown only with JavaScript", () => {
    page();
    const button = screen.getByRole("button", { name: "Tastenkürzel" });
    expect(button).toBeVisible();
    fireEvent.click(button);
    expect(screen.getByRole("dialog", { name: "Tastenkürzel" })).toBeInTheDocument();
  });

  it("also moves through the tiles of the view by source", () => {
    render(
      <>
        <section className="tile">
          <a className="title" href="https://example.com/a">
            Kachel A
          </a>
        </section>
        <Shortcuts />
      </>,
    );
    press("j");
    expect(focused()).toBe("Kachel A");
  });
});
