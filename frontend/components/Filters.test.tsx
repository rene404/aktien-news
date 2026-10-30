import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Source } from "@/lib/api";
import Filters from "./Filters";

const sources: Source[] = [
  { key: "fn_adhoc", name: "FinanzNachrichten Ad-hoc", market: "MULTI", kind: "adhoc" },
  { key: "fn_atx", name: "FinanzNachrichten ATX", market: "AT", kind: "news_api" },
];

describe("Filters", () => {
  it("is a GET form to the page itself, so it works without JavaScript", () => {
    const { container } = render(<Filters path="/aktie/EBS" sources={sources} value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    const form = container.querySelector("form")!;
    expect(form).toHaveAttribute("action", "/aktie/EBS");
    expect(form.method).toBe("get");
    expect(screen.getByRole("button", { name: /filtern/i })).toHaveAttribute("type", "submit");
  });

  it("applies a changed select at once", () => {
    const { container } = render(<Filters path="/" sources={sources} value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    const onSubmit = vi.fn((e: Event) => e.preventDefault());
    container.querySelector("form")!.addEventListener("submit", onSubmit);
    fireEvent.change(screen.getByLabelText("Zeitraum"), { target: { value: "24h" } });
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("preselects the current values and keeps the section", () => {
    const { container } = render(<Filters path="/" sources={sources} value={{ section: "ATX", source: "fn_atx", range: "24h", cursor: "" }} />);
    expect(screen.getByLabelText("Quelle")).toHaveValue("fn_atx");
    expect(screen.getByLabelText("Zeitraum")).toHaveValue("24h");
    expect(container.querySelector('input[name="section"]')).toHaveValue("ATX");
  });

  it("offers every source plus 'Alle'", () => {
    render(<Filters path="/" sources={sources} value={{ section: "", source: "", range: "7d", cursor: "" }} />);
    const options = Array.from((screen.getByLabelText("Quelle") as HTMLSelectElement).options).map((o) => o.textContent);
    expect(options).toEqual(["Alle", "FinanzNachrichten Ad-hoc", "FinanzNachrichten ATX"]);
  });

  it("never carries the pagination cursor into a new filter", () => {
    const { container } = render(<Filters path="/" sources={sources} value={{ section: "", source: "", range: "7d", cursor: "abc" }} />);
    expect(container.querySelector('input[name="cursor"]')).toBeNull();
  });
});
