import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import ViewSwitch from "./ViewSwitch";

describe("ViewSwitch", () => {
  it("switches between the feed and the view by source, keeping section and range", () => {
    render(<ViewSwitch current="feed" value={{ section: "DAX", source: "fn_dax", range: "24h", cursor: "c1" }} />);
    expect(screen.getByRole("navigation", { name: "Ansicht" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Feed" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Feed" })).toHaveAttribute("href", "/?section=DAX&range=24h");
    expect(screen.getByRole("link", { name: "Nach Quelle" })).toHaveAttribute("href", "/quellen?section=DAX&range=24h");
    expect(screen.getByRole("link", { name: "Nach Quelle" })).not.toHaveAttribute("aria-current");
  });
});
