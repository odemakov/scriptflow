// File: src/lib/helpers.ts
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { Seconds2human, TimeAgo, TimeDiff } from "@/lib/helpers";

describe("Seconds2human", () => {
  it('should return "0" for input 0', () => {
    expect(Seconds2human(0)).toBe("0");
  });

  it("should return seconds only if < 60s", () => {
    expect(Seconds2human(1)).toBe("1s");
    expect(Seconds2human(30)).toBe("30s");
    expect(Seconds2human(59)).toBe("59s");
  });

  it("should return minutes+seconds if >= 60s && < 5m", () => {
    expect(Seconds2human(60)).toBe("1m");
    expect(Seconds2human(90)).toBe("1m30s");
    expect(Seconds2human(210)).toBe("3m30s");
    expect(Seconds2human(299)).toBe("4m59s");
  });

  it("should return minutes only if >= 5m && < 60m", () => {
    expect(Seconds2human(300)).toBe("5m");
    expect(Seconds2human(1800)).toBe("30m");
    expect(Seconds2human(3540)).toBe("59m");
  });

  it("should return hours+minutes if >= 60m && < 4h", () => {
    expect(Seconds2human(3600)).toBe("1h");
    expect(Seconds2human(9000)).toBe("2h30m");
    expect(Seconds2human(14340)).toBe("3h59m");
  });

  it("should return hours only if >= 4h && < 1d", () => {
    expect(Seconds2human(14400)).toBe("4h");
    expect(Seconds2human(43200)).toBe("12h");
    expect(Seconds2human(82800)).toBe("23h");
  });

  it("should return days+hours if >= 1d && < 4d", () => {
    expect(Seconds2human(86400)).toBe("1d");
    expect(Seconds2human(216000)).toBe("2d12h");
    expect(Seconds2human(342000)).toBe("3d23h");
  });

  it("should return days only if >= 4d && < 60d", () => {
    expect(Seconds2human(345600)).toBe("4d");
    expect(Seconds2human(2592000)).toBe("30d");
    expect(Seconds2human(5097600)).toBe("59d");
  });

  it("should return months only if >= 60d && < 365d", () => {
    expect(Seconds2human(5184000)).toBe("2mo");
    expect(Seconds2human(15552000)).toBe("6mo");
    expect(Seconds2human(31104000)).toBe("12mo");
  });

  it("should return years+months if >= 365d", () => {
    expect(Seconds2human(31536000)).toBe("1y");
    expect(Seconds2human(78840000)).toBe("2y6mo");
    expect(Seconds2human(323136000)).toBe("10y3mo");
  });

  it("should handle zero remainders correctly", () => {
    expect(Seconds2human(60)).toBe("1m");
    expect(Seconds2human(3600)).toBe("1h");
    expect(Seconds2human(86400)).toBe("1d");
    expect(Seconds2human(31536000)).toBe("1y");
  });
});

describe("TimeAgo", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return time elapsed from past datetime", () => {
    const now = new Date("2025-01-15T12:00:00Z");
    vi.setSystemTime(now);

    expect(TimeAgo("2025-01-15T11:59:30Z")).toBe("30s");
    expect(TimeAgo("2025-01-15T11:58:00Z")).toBe("2m");
    expect(TimeAgo("2025-01-15T10:30:00Z")).toBe("1h30m");
    expect(TimeAgo("2025-01-14T12:00:00Z")).toBe("1d");
  });

  it("should handle future dates (clock skew) by clamping to 0", () => {
    const now = new Date("2025-01-15T12:00:00Z");
    vi.setSystemTime(now);

    expect(TimeAgo("2025-01-15T13:00:00Z")).toBe("0");
    expect(TimeAgo("2025-01-15T12:00:05Z")).toBe("0");
  });

  it("should round seconds properly", () => {
    const now = new Date("2025-01-15T12:00:00.999Z");
    vi.setSystemTime(now);

    expect(TimeAgo("2025-01-15T11:59:30.500Z")).toBe("30s");
  });
});

describe("TimeDiff", () => {
  it("should calculate difference between two datetimes", () => {
    expect(TimeDiff("2025-01-15T12:00:00Z", "2025-01-15T12:00:30Z")).toBe("30s");
    expect(TimeDiff("2025-01-15T12:00:00Z", "2025-01-15T12:02:00Z")).toBe("2m");
    expect(TimeDiff("2025-01-15T12:00:00Z", "2025-01-15T13:30:00Z")).toBe("1h30m");
    expect(TimeDiff("2025-01-15T12:00:00Z", "2025-01-16T12:00:00Z")).toBe("1d");
  });

  it("should handle negative differences (end before start) by clamping to 0", () => {
    expect(TimeDiff("2025-01-15T13:00:00Z", "2025-01-15T12:00:00Z")).toBe("0");
  });

  it("should handle same timestamps", () => {
    expect(TimeDiff("2025-01-15T12:00:00Z", "2025-01-15T12:00:00Z")).toBe("0");
  });

  it("should round milliseconds properly", () => {
    expect(TimeDiff("2025-01-15T12:00:00.000Z", "2025-01-15T12:00:30.999Z")).toBe("31s");
  });
});
