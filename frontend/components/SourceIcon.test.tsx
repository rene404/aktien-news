import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import SourceIcon from "./SourceIcon";

describe("SourceIcon", () => {
  it("shows the publisher's icon, served by us, as decoration next to the name", () => {
    const { container } = render(<SourceIcon sourceKey="fn_dax" name="FinanzNachrichten DAX" />);
    const img = container.querySelector("img")!;
    expect(img).toHaveAttribute("src", "/sources/fn.png");
    expect(img).toHaveAttribute("alt", ""); // the name follows as text
    expect(img).toHaveAttribute("width", "16");
    expect(img).toHaveAttribute("loading", "lazy");
  });

  it("falls back to the first letter of the name, drawn by CSS, so it adds no text", () => {
    const { container } = render(<SourceIcon sourceKey="borsa_news" name="Borsa News" />);
    expect(container.querySelector("img")).toBeNull();
    const letter = container.querySelector(".src-icon")!;
    expect(letter).toHaveAttribute("data-letter", "B");
    expect(letter).toHaveAttribute("aria-hidden", "true");
    expect(letter.textContent).toBe("");
  });
});
