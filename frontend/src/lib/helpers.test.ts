// File: src/lib/helpers.ts
import { describe, it, expect } from 'vitest';
import { Seconds2human } from '@/lib/helpers';

describe('Seconds2human', () => {
  it('should return "0" for input 0', () => {
    expect(Seconds2human(0)).toBe("0");
  });

  it('should return seconds if input is less than 60', () => {
    expect(Seconds2human(10)).toBe("10s");
    expect(Seconds2human(59)).toBe("59s");
  });

  it('should return minutes and seconds if input is less than 600', () => {
    expect(Seconds2human(120)).toBe("2m0s");
    expect(Seconds2human(146)).toBe("2m26s");
    expect(Seconds2human(599)).toBe("9m59s");
  });

  it('should return minutes if input is less than 3600 and greater than or equal to 600', () => {
    expect(Seconds2human(600)).toBe("10m");
    expect(Seconds2human(3599)).toBe("59m");
  });

  it('should return hours and minutes if input is 3600 or more', () => {
    expect(Seconds2human(3600)).toBe("1h");
    expect(Seconds2human(3660)).toBe("1h1");
    expect(Seconds2human(7260)).toBe("2h1");
  });

  it('should return only hours if minutes are 0', () => {
    expect(Seconds2human(7200)).toBe("2h");
  });

  it('should handle large inputs correctly', () => {
    expect(Seconds2human(86400)).toBe("24h");
    expect(Seconds2human(90000)).toBe("25h");
  });

  it('should handle edge cases', () => {
    expect(Seconds2human(1)).toBe("1s");
    expect(Seconds2human(599)).toBe("9m59s");
    expect(Seconds2human(3599)).toBe("59m");
    expect(Seconds2human(3600)).toBe("1h");
  });
});

