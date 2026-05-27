package model

import "strings"

type countryEntry struct {
	name string
	flag string
}

var countryByCode = map[string]countryEntry{
	"DE": {name: "Германия", flag: "🇩🇪"},
	"US": {name: "США", flag: "🇺🇸"},
	"GB": {name: "Великобритания", flag: "🇬🇧"},
	"UK": {name: "Великобритания", flag: "🇬🇧"},
	"NL": {name: "Нидерланды", flag: "🇳🇱"},
	"FR": {name: "Франция", flag: "🇫🇷"},
	"FI": {name: "Финляндия", flag: "🇫🇮"},
	"SE": {name: "Швеция", flag: "🇸🇪"},
	"NO": {name: "Норвегия", flag: "🇳🇴"},
	"CH": {name: "Швейцария", flag: "🇨🇭"},
	"AT": {name: "Австрия", flag: "🇦🇹"},
	"PL": {name: "Польша", flag: "🇵🇱"},
	"CZ": {name: "Чехия", flag: "🇨🇿"},
	"RO": {name: "Румыния", flag: "🇷🇴"},
	"BG": {name: "Болгария", flag: "🇧🇬"},
	"UA": {name: "Украина", flag: "🇺🇦"},
	"RU": {name: "Россия", flag: "🇷🇺"},
	"KZ": {name: "Казахстан", flag: "🇰🇿"},
	"TR": {name: "Турция", flag: "🇹🇷"},
	"JP": {name: "Япония", flag: "🇯🇵"},
	"KR": {name: "Корея", flag: "🇰🇷"},
	"SG": {name: "Сингапур", flag: "🇸🇬"},
	"HK": {name: "Гонконг", flag: "🇭🇰"},
	"TW": {name: "Тайвань", flag: "🇹🇼"},
	"CA": {name: "Канада", flag: "🇨🇦"},
	"AU": {name: "Австралия", flag: "🇦🇺"},
	"BR": {name: "Бразилия", flag: "🇧🇷"},
	"IN": {name: "Индия", flag: "🇮🇳"},
	"IL": {name: "Израиль", flag: "🇮🇱"},
	"AE": {name: "ОАЭ", flag: "🇦🇪"},
	"LT": {name: "Литва", flag: "🇱🇹"},
	"LV": {name: "Латвия", flag: "🇱🇻"},
	"EE": {name: "Эстония", flag: "🇪🇪"},
	"IT": {name: "Италия", flag: "🇮🇹"},
	"ES": {name: "Испания", flag: "🇪🇸"},
	"PT": {name: "Португалия", flag: "🇵🇹"},
	"IE": {name: "Ирландия", flag: "🇮🇪"},
	"DK": {name: "Дания", flag: "🇩🇰"},
	"LU": {name: "Люксембург", flag: "🇱🇺"},
	"IS": {name: "Исландия", flag: "🇮🇸"},
}

var nameAliases = map[string]string{
	"germany": "DE", "deutschland": "DE", "германия": "DE",
	"usa": "US", "united states": "US", "сша": "US", "america": "US",
	"netherlands": "NL", "holland": "NL", "нидерланды": "NL",
	"finland": "FI", "финляндия": "FI",
	"sweden": "SE", "швеция": "SE",
	"norway": "NO", "норвегия": "NO",
	"switzerland": "CH", "швейцария": "CH",
	"france": "FR", "франция": "FR",
	"poland": "PL", "польша": "PL",
	"japan": "JP", "япония": "JP",
	"singapore": "SG", "сингапур": "SG",
	"canada": "CA", "канада": "CA",
	"australia": "AU", "австралия": "AU",
	"russia": "RU", "россия": "RU",
	"uk": "GB", "britain": "GB", "united kingdom": "GB", "великобритания": "GB", "england": "GB",
}

// FlagEmojiForCode returns a flag emoji for ISO 3166-1 alpha-2.
func FlagEmojiForCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if e, ok := countryByCode[c]; ok {
		return e.flag
	}
	if len(c) != 2 {
		return "🌐"
	}
	a := 0x1F1E6 + int(c[0]-'A')
	b := 0x1F1E6 + int(c[1]-'A')
	return string([]rune{rune(a), rune(b)})
}

// ResolveCountryToken maps ISO code or country name to display fields.
func ResolveCountryToken(token string) (code, name, flag string, ok bool) {
	t := strings.TrimSpace(token)
	if t == "" {
		return "", "", "", false
	}
	upper := strings.ToUpper(t)
	if len(upper) == 2 {
		if e, ok := countryByCode[upper]; ok {
			return upper, e.name, e.flag, true
		}
	}
	if code, ok := nameAliases[strings.ToLower(t)]; ok {
		if e, ok2 := countryByCode[code]; ok2 {
			return code, e.name, e.flag, true
		}
	}
	return "", "", "", false
}

// ExtractFlagEmoji splits a leading regional-indicator flag from s.
func ExtractFlagEmoji(s string) (flag, rest string) {
	runes := []rune(s)
	for i := 0; i+1 < len(runes); i++ {
		if isRegionalIndicator(runes[i]) && isRegionalIndicator(runes[i+1]) {
			flag = string(runes[i : i+2])
			rest = strings.TrimSpace(string(runes[:i]) + string(runes[i+2:]))
			return flag, rest
		}
	}
	return "", s
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}
