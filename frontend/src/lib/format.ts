import { extractFlagEmoji, flagEmojiForCode, resolveCountryToken } from "./flags";

export type ServerDisplay = {
  flag: string;
  country: string;
  serverName: string;
  showCountry: boolean;
};

const SPLIT_RE = /[|·•–—_/]+/;

export function formatBps(bytesPerSec: number): string {
  if (bytesPerSec <= 0) return "0 B/s";
  const units = ["B/s", "KB/s", "MB/s", "GB/s"];
  let v = bytesPerSec;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = i === 0 ? 0 : v >= 100 ? 0 : 1;
  return `${v.toFixed(digits)} ${units[i]}`;
}

export function formatDuration(totalSec: number): string {
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  return [h, m, s].map((n) => String(n).padStart(2, "0")).join(":");
}

function norm(s: string): string {
  return s.trim().replace(/\s+/g, " ");
}

function stripPrefix(rest: string, token: string): string {
  let r = rest.trim();
  const low = r.toLowerCase();
  const t = token.toLowerCase();
  if (low.startsWith(t)) {
    r = r.slice(token.length).replace(/^[\s|·•\-–—_]+/, "").trim();
  }
  return r;
}

function parseTokens(parts: string[]): {
  flag: string;
  country: string;
  code: string;
  serverName: string;
} {
  let flag = "";
  let country = "";
  let code = "";
  const nameParts: string[] = [];

  for (const raw of parts) {
    const part = norm(raw);
    if (!part) continue;
    const { flag: f, rest } = extractFlagEmoji(part);
    const work = f ? rest : part;
    if (f && !flag) flag = f;

    const iso = work.match(/\b([A-Za-z]{2})\b/);
    if (iso) {
      const resolved = resolveCountryToken(iso[1]);
      if (resolved) {
        if (!flag) flag = resolved.flag;
        if (!country) country = resolved.name;
        if (!code) code = resolved.code;
        const leftover = work.replace(iso[0], "").trim();
        if (leftover) nameParts.push(leftover);
        continue;
      }
    }

    const resolved = resolveCountryToken(work);
    if (resolved && work.length <= 24) {
      if (!flag) flag = resolved.flag;
      if (!country) country = resolved.name;
      if (!code) code = resolved.code;
      continue;
    }
    nameParts.push(work);
  }

  let serverName = norm(nameParts.join(" "));
  if (country) serverName = stripPrefix(serverName, country);
  if (code) serverName = stripPrefix(serverName, code);
  if (flag) serverName = stripPrefix(serverName, flag);
  serverName = norm(serverName);

  if (!flag && code) flag = flagEmojiForCode(code);
  return { flag, country, code, serverName };
}

/** Parse subscription profile name into flag, country, server title without duplicates. */
export function parseServerDisplay(rawName: string): ServerDisplay {
  const raw = norm(rawName);
  if (!raw) {
    return { flag: "🌐", country: "", serverName: "", showCountry: false };
  }

  const { flag: leadFlag, rest: afterFlag } = extractFlagEmoji(raw);
  const parts = (afterFlag || raw).split(SPLIT_RE).map(norm).filter(Boolean);
  const parsed = parseTokens(leadFlag ? [leadFlag, ...parts] : parts);

  let { flag, country, serverName } = parsed;
  if (!serverName) {
    serverName = afterFlag || raw;
    country = "";
  }
  if (!flag) flag = "🌐";

  const showCountry =
    country !== "" &&
    !serverName.toLowerCase().includes(country.toLowerCase()) &&
    country.toLowerCase() !== serverName.toLowerCase();

  return { flag, country, serverName, showCountry };
}

export function disconnectedServerTitle(profileName: string): ServerDisplay {
  if (!profileName.trim()) {
    return {
      flag: "🌐",
      country: "",
      serverName: "Автовыбор",
      showCountry: false,
    };
  }
  const d = parseServerDisplay(profileName);
  return { ...d, serverName: d.serverName || profileName };
}
