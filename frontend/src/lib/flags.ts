/** ISO 3166-1 alpha-2 → display name (RU) and flag emoji. */
const COUNTRY: Record<string, { name: string; flag: string }> = {
  DE: { name: "Германия", flag: "🇩🇪" },
  US: { name: "США", flag: "🇺🇸" },
  GB: { name: "Великобритания", flag: "🇬🇧" },
  UK: { name: "Великобритания", flag: "🇬🇧" },
  NL: { name: "Нидерланды", flag: "🇳🇱" },
  FR: { name: "Франция", flag: "🇫🇷" },
  FI: { name: "Финляндия", flag: "🇫🇮" },
  SE: { name: "Швеция", flag: "🇸🇪" },
  NO: { name: "Норвегия", flag: "🇳🇴" },
  CH: { name: "Швейцария", flag: "🇨🇭" },
  AT: { name: "Австрия", flag: "🇦🇹" },
  PL: { name: "Польша", flag: "🇵🇱" },
  CZ: { name: "Чехия", flag: "🇨🇿" },
  RO: { name: "Румыния", flag: "🇷🇴" },
  BG: { name: "Болгария", flag: "🇧🇬" },
  UA: { name: "Украина", flag: "🇺🇦" },
  RU: { name: "Россия", flag: "🇷🇺" },
  KZ: { name: "Казахстан", flag: "🇰🇿" },
  TR: { name: "Турция", flag: "🇹🇷" },
  JP: { name: "Япония", flag: "🇯🇵" },
  KR: { name: "Корея", flag: "🇰🇷" },
  SG: { name: "Сингапур", flag: "🇸🇬" },
  HK: { name: "Гонконг", flag: "🇭🇰" },
  TW: { name: "Тайвань", flag: "🇹🇼" },
  CA: { name: "Канада", flag: "🇨🇦" },
  AU: { name: "Австралия", flag: "🇦🇺" },
  BR: { name: "Бразилия", flag: "🇧🇷" },
  IN: { name: "Индия", flag: "🇮🇳" },
  IL: { name: "Израиль", flag: "🇮🇱" },
  AE: { name: "ОАЭ", flag: "🇦🇪" },
  LT: { name: "Литва", flag: "🇱🇹" },
  LV: { name: "Латвия", flag: "🇱🇻" },
  EE: { name: "Эстония", flag: "🇪🇪" },
  IT: { name: "Италия", flag: "🇮🇹" },
  ES: { name: "Испания", flag: "🇪🇸" },
  PT: { name: "Португалия", flag: "🇵🇹" },
  IE: { name: "Ирландия", flag: "🇮🇪" },
  DK: { name: "Дания", flag: "🇩🇰" },
  LU: { name: "Люксембург", flag: "🇱🇺" },
  IS: { name: "Исландия", flag: "🇮🇸" },
};

const NAME_ALIASES: Record<string, string> = {
  germany: "DE",
  deutschland: "DE",
  германия: "DE",
  usa: "US",
  "united states": "US",
  сша: "US",
  america: "US",
  netherlands: "NL",
  holland: "NL",
  нидерланды: "NL",
  finland: "FI",
  финляндия: "FI",
  sweden: "SE",
  швеция: "SE",
  norway: "NO",
  норвегия: "NO",
  switzerland: "CH",
  швейцария: "CH",
  france: "FR",
  франция: "FR",
  poland: "PL",
  польша: "PL",
  japan: "JP",
  япония: "JP",
  singapore: "SG",
  сингапур: "SG",
  canada: "CA",
  канада: "CA",
  australia: "AU",
  австралия: "AU",
  russia: "RU",
  россия: "RU",
  uk: "GB",
  britain: "GB",
  "united kingdom": "GB",
  великобритания: "GB",
  england: "GB",
};

export function flagEmojiForCode(code: string): string {
  const c = code.toUpperCase();
  const entry = COUNTRY[c];
  if (entry) return entry.flag;
  if (c.length !== 2) return "🌐";
  const a = 0x1f1e6 + (c.charCodeAt(0) - 65);
  const b = 0x1f1e6 + (c.charCodeAt(1) - 65);
  return String.fromCodePoint(a, b);
}

export function countryNameForCode(code: string): string {
  const c = code.toUpperCase();
  return COUNTRY[c]?.name ?? code;
}

export function resolveCountryToken(token: string): { code: string; name: string; flag: string } | null {
  const t = token.trim();
  if (!t) return null;
  const upper = t.toUpperCase();
  if (upper.length === 2 && COUNTRY[upper]) {
    return { code: upper, name: COUNTRY[upper].name, flag: COUNTRY[upper].flag };
  }
  const key = t.toLowerCase();
  const code = NAME_ALIASES[key];
  if (code && COUNTRY[code]) {
    return { code, name: COUNTRY[code].name, flag: COUNTRY[code].flag };
  }
  return null;
}

export function extractFlagEmoji(s: string): { flag: string; rest: string } {
  const re = /\p{Regional_Indicator}{2}/u;
  const m = s.match(re);
  if (!m || m.index === undefined) return { flag: "", rest: s };
  const flag = m[0];
  const rest = (s.slice(0, m.index) + s.slice(m.index + flag.length)).trim();
  return { flag, rest };
}
