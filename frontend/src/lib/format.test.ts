import { describe, expect, it } from "vitest";
import { formatBps, formatDuration, parseServerDisplay } from "./format";

describe("parseServerDisplay", () => {
  it("parses emoji + code + city", () => {
    const d = parseServerDisplay("🇩🇪 DE Frankfurt #1");
    expect(d.flag).toBe("🇩🇪");
    expect(d.country).toBe("Германия");
    expect(d.serverName).toBe("Frankfurt #1");
    expect(d.showCountry).toBe(true);
  });

  it("parses country dot city", () => {
    const d = parseServerDisplay("Germany · Frankfurt");
    expect(d.country).toBe("Германия");
    expect(d.serverName).toBe("Frankfurt");
  });

  it("parses US-NY code prefix", () => {
    const d = parseServerDisplay("US-NY-01");
    expect(d.country).toBe("США");
    expect(d.serverName).toContain("NY");
  });

  it("falls back for geo-less node", () => {
    const d = parseServerDisplay("Node-42");
    expect(d.serverName).toBe("Node-42");
    expect(d.showCountry).toBe(false);
  });
});

describe("formatBps", () => {
  it("formats megabytes", () => {
    expect(formatBps(12.4 * 1024 * 1024)).toMatch(/MB\/s/);
  });
});

describe("formatDuration", () => {
  it("pads hours", () => {
    expect(formatDuration(255)).toBe("00:04:15");
  });
});
