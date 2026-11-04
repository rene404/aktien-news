import { describe, expect, it } from "vitest";
import { dayKey, dayLabel, formatClock, formatTime } from "./format";

describe("format", () => {
  it("shows Vienna time", () => {
    expect(formatTime("2026-10-28T09:30:00Z")).toBe("28.10.2026, 10:30");
    expect(formatClock("2026-10-28T09:30:00Z")).toBe("10:30");
  });

  it("groups by the Vienna calendar day", () => {
    expect(dayKey("2026-10-28T23:30:00Z")).toBe("2026-10-29"); // 00:30 in Vienna
  });

  it("names today and yesterday, spells out older days", () => {
    const now = new Date("2026-10-29T10:00:00Z");
    expect(dayLabel("2026-10-29", now)).toBe("Heute");
    expect(dayLabel("2026-10-28", now)).toBe("Gestern");
    expect(dayLabel("2026-10-26", now)).toBe("Montag, 26.10.2026");
  });

  // Regression: "Gestern" was now − 24 h, which misses on the 23- and 25-hour days.
  it("finds yesterday on the days the clocks change", () => {
    // 2026-10-25 has 25 hours; 23:30 CET.
    expect(dayLabel("2026-10-24", new Date("2026-10-25T22:30:00Z"))).toBe("Gestern");
    // 2026-03-29 has 23 hours; 2026-03-30 00:30 CEST.
    expect(dayLabel("2026-03-29", new Date("2026-03-29T22:30:00Z"))).toBe("Gestern");
    expect(dayLabel("2026-03-28", new Date("2026-03-29T22:30:00Z"))).toBe("Samstag, 28.03.2026");
  });
});
