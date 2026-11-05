import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import TickerChips from "./TickerChips";

const items = [
  { ticker: "EBS", name: "Erste Group", count: 3 },
  { ticker: "POST", name: "Österreichische Post", count: 0 },
];

describe("TickerChips", () => {
  it("links each stock to its page with name, ticker and count", () => {
    render(<TickerChips items={items} label="ATX-Aktien" />);
    const link = screen.getByRole("link", { name: /Erste Group/ });
    expect(link).toHaveAttribute("href", "/aktie/EBS");
    expect(link).toHaveTextContent("Erste Group");
    expect(link).toHaveTextContent("EBS");
    expect(link).toHaveTextContent("3");
    expect(link).toHaveAccessibleName("Erste Group (EBS), 3 Meldungen");
  });

  it("dims stocks without news", () => {
    render(<TickerChips items={items} label="ATX-Aktien" />);
    expect(screen.getByRole("link", { name: /Post/ })).toHaveClass("zero");
    expect(screen.getByRole("link", { name: /Post/ })).toHaveAccessibleName("Österreichische Post (POST), keine Meldungen");
    expect(screen.getByRole("link", { name: /Erste/ })).not.toHaveClass("zero");
  });

  it("can leave out the name for a compact row", () => {
    render(<TickerChips items={items} label="Heute im Gespräch" compact />);
    expect(screen.getByRole("link", { name: /Erste Group/ })).not.toHaveTextContent("Erste Group");
    expect(screen.getByRole("list", { name: "Heute im Gespräch" })).toBeInTheDocument();
  });

  it("singular for one", () => {
    render(<TickerChips items={[{ ticker: "OMV", name: "OMV", count: 1 }]} label="x" />);
    expect(screen.getByRole("link")).toHaveAccessibleName("OMV (OMV), 1 Meldung");
  });
});
