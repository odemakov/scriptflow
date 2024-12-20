// File: src/lib/helpers.ts
import { describe, it, expect } from "vitest";
import { Seconds2human } from "@/lib/helpers";

describe("Seconds2human", () => {
  it('should return "0" for input 0', () => {
    expect(Seconds2human(0)).toBe("0");
  });

  it("should return seconds if input is less than 60", () => {
    expect(Seconds2human(10)).toBe("10s");
    expect(Seconds2human(59)).toBe("59s");
  });

  it("should return minutes and seconds if input is less than 3600", () => {
    expect(Seconds2human(120)).toBe("2m");
    expect(Seconds2human(146)).toBe("2m26s");
    expect(Seconds2human(599)).toBe("9m59s");
    expect(Seconds2human(600)).toBe("10m");
    expect(Seconds2human(1800)).toBe("30m");
    expect(Seconds2human(3599)).toBe("59m59s");
  });

  it("should return hours and minutes if input is less than 3600*24", () => {
    expect(Seconds2human(3600)).toBe("1h");
    expect(Seconds2human(3660)).toBe("1h1m");
    expect(Seconds2human(7260)).toBe("2h1m");
  });

  it("should return days and hours if input is 3600 or more", () => {
    expect(Seconds2human(3600 * 24)).toBe("1d");
    expect(Seconds2human(3600 * 11 + 70)).toBe("11h1m");
    expect(Seconds2human(3600 * 20 + 662)).toBe("20h11m");
  });

  it("should return only minutes if seconds are 0", () => {
    expect(Seconds2human(120)).toBe("2m");
  });
  it("should return only hours if minutes are 0", () => {
    expect(Seconds2human(7200)).toBe("2h");
  });
  it("should return only days if hours are 0", () => {
    expect(Seconds2human(24 * 3600 * 3)).toBe("3d");
  });

  it("should handle large inputs correctly", () => {
    expect(Seconds2human(86400)).toBe("1d");
    expect(Seconds2human(90000)).toBe("1d1h");
  });

  it("should handle edge cases", () => {
    expect(Seconds2human(1)).toBe("1s");
    expect(Seconds2human(599)).toBe("9m59s");
    expect(Seconds2human(3599)).toBe("59m59s");
    expect(Seconds2human(3600)).toBe("1h");
  });
});
