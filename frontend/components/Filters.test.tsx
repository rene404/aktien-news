import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import Filters from "./Filters";

const sources = [
  { key: "fn_adhoc", name: "FinanzNachrichten.de Ad hoc", market: "MULTI" },
  { key: "fn_atx", name: "FinanzNachrichten.de ATX", market: "AT" },
];

describe("Filters", () => {
  it("is a plain GET form, so it works without JavaScript", () => {
    const { container } = render(<Filters sources={sources} value={{ index: "", source: "", range: "7d", cursor: "" }} />);
    const form = container.querySelector("form");
    expect(form).toHaveAttribute("method", "get");
    expect(screen.getByRole("button", { name: /filtern/i })).toHaveAttribute("type", "submit");
  });

  it("preselects the current values", () => {
    render(<Filters sources={sources} value={{ index: "ATX", source: "fn_atx", range: "24h", cursor: "" }} />);
    expect(screen.getByLabelText("Index")).toHaveValue("ATX");
    expect(screen.getByLabelText("Quelle")).toHaveValue("fn_atx");
    expect(screen.getByLabelText("Zeitraum")).toHaveValue("24h");
  });

  it("offers every source plus 'Alle'", () => {
    render(<Filters sources={sources} value={{ index: "", source: "", range: "7d", cursor: "" }} />);
    const options = Array.from((screen.getByLabelText("Quelle") as HTMLSelectElement).options).map((o) => o.textContent);
    expect(options).toEqual(["Alle", "FinanzNachrichten.de Ad hoc", "FinanzNachrichten.de ATX"]);
  });

  it("never carries the pagination cursor into a new filter", () => {
    const { container } = render(<Filters sources={sources} value={{ index: "", source: "", range: "7d", cursor: "abc" }} />);
    expect(container.querySelector('input[name="cursor"]')).toBeNull();
  });
});
