import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Source } from "@/lib/api";
import Filters from "./Filters";

const sources: Source[] = [
  { key: "fn_adhoc", name: "FinanzNachrichten Ad-hoc", market: "MULTI", kind: "adhoc" },
  { key: "fn_atx", name: "FinanzNachrichten ATX", market: "AT", kind: "news_api" },
];

describe("Filters", () => {
  it("is a plain GET form, so it works without JavaScript", () => {
    const { container } = render(<Filters sources={sources} value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    const form = container.querySelector("form");
    expect(form).toHaveAttribute("method", "get");
    expect(screen.getByRole("button", { name: /filtern/i })).toHaveAttribute("type", "submit");
  });

  it("preselects the current values and keeps the section", () => {
    const { container } = render(<Filters sources={sources} value={{ section: "ATX", source: "fn_atx", range: "24h", cursor: "" }} />);
    expect(screen.getByLabelText("Quelle")).toHaveValue("fn_atx");
    expect(screen.getByLabelText("Zeitraum")).toHaveValue("24h");
    expect(container.querySelector('input[name="section"]')).toHaveValue("ATX");
  });

  it("offers every source plus 'Alle'", () => {
    render(<Filters sources={sources} value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    const options = Array.from((screen.getByLabelText("Quelle") as HTMLSelectElement).options).map((o) => o.textContent);
    expect(options).toEqual(["Alle", "FinanzNachrichten Ad-hoc", "FinanzNachrichten ATX"]);
  });

  it("never carries the pagination cursor into a new filter", () => {
    const { container } = render(<Filters sources={sources} value={{ section: "", source: "", range: "7d", cursor: "abc" }} />);
    expect(container.querySelector('input[name="cursor"]')).toBeNull();
  });
});
