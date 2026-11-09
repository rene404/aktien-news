import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import SectionTabs from "./SectionTabs";

describe("SectionTabs", () => {
  it("links every section, marks the current one and keeps the range", () => {
    render(<SectionTabs value={{ section: "ATX", source: "fn_atx", range: "24h", cursor: "c1" }} />);
    const links = screen.getAllByRole("link");
    expect(links.map((l) => l.textContent)).toEqual(["Alle", "ATX", "DAX", "CAC 40", "IBEX 35", "AEX", "OMXS30", "OMXH25", "OMXC25", "FTSE 100", "S&P 500", "Ad-hoc", "Pflichtmeldungen"]);
    expect(screen.getByRole("link", { name: "ATX" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Alle" })).not.toHaveAttribute("aria-current");
    expect(screen.getByRole("link", { name: "Pflichtmeldungen" })).toHaveAttribute("href", "/?section=filings&range=24h");
  });

  it("can link the sections of another page", () => {
    render(<SectionTabs path="/quellen" value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    expect(screen.getByRole("link", { name: "DAX" })).toHaveAttribute("href", "/quellen?section=DAX");
  });
});
