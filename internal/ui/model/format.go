package model

import (
	"fmt"
	"regexp"
	"strings"
)

// ServerDisplay is parsed profile/server label for UI cards.
type ServerDisplay struct {
	Flag        string
	Country     string
	ServerName  string
	ShowCountry bool
}

var (
	splitRE   = regexp.MustCompile(`[|·•–—_/]+`) // no ASCII hyphen (keeps US-NY-01)
	isoCodeRE = regexp.MustCompile(`\b([A-Za-z]{2})\b`)
)

// FormatBps formats bytes per second for traffic rows.
func FormatBps(bytesPerSec int64) string {
	if bytesPerSec <= 0 {
		return "0 B/s"
	}
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	v := float64(bytesPerSec)
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	digits := 0
	if i > 0 && v < 100 {
		digits = 1
	}
	return fmt.Sprintf("%.*f %s", digits, v, units[i])
}

// FormatDuration formats seconds as HH:MM:SS.
func FormatDuration(totalSec int) string {
	if totalSec < 0 {
		totalSec = 0
	}
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// ParseServerDisplay splits subscription profile name into flag, country, server title.
func ParseServerDisplay(rawName string) ServerDisplay {
	raw := normSpace(rawName)
	if raw == "" {
		return ServerDisplay{Flag: "🌐", ServerName: "", ShowCountry: false}
	}
	leadFlag, afterFlag := ExtractFlagEmoji(raw)
	parts := splitRE.Split(afterFlag, -1)
	if afterFlag == "" {
		parts = splitRE.Split(raw, -1)
	}
	var clean []string
	for _, p := range parts {
		if p = normSpace(p); p != "" {
			clean = append(clean, p)
		}
	}
	tokens := clean
	if leadFlag != "" {
		tokens = append([]string{leadFlag}, clean...)
	}
	parsed := parseTokens(tokens)
	flag, country, serverName := parsed.flag, parsed.country, parsed.serverName
	if serverName == "" {
		serverName = afterFlag
		if serverName == "" {
			serverName = raw
		}
		country = ""
	}
	if flag == "" {
		flag = "🌐"
	}
	show := country != "" &&
		!strings.Contains(strings.ToLower(serverName), strings.ToLower(country)) &&
		!strings.EqualFold(country, serverName)
	return ServerDisplay{Flag: flag, Country: country, ServerName: serverName, ShowCountry: show}
}

// DisconnectedServerTitle picks card title when VPN is off.
func DisconnectedServerTitle(profileName string) ServerDisplay {
	if strings.TrimSpace(profileName) == "" {
		return ServerDisplay{Flag: "🌐", ServerName: "Автовыбор", ShowCountry: false}
	}
	d := ParseServerDisplay(profileName)
	if d.ServerName == "" {
		d.ServerName = profileName
	}
	return d
}

type parsedTokens struct {
	flag, country, code, serverName string
}

func parseTokens(parts []string) parsedTokens {
	var out parsedTokens
	var nameParts []string
	for _, raw := range parts {
		part := normSpace(raw)
		if part == "" {
			continue
		}
		f, work := ExtractFlagEmoji(part)
		if f != "" && out.flag == "" {
			out.flag = f
		}
		if work == "" {
			work = part
		}
		if iso := isoCodeRE.FindStringSubmatch(work); len(iso) == 2 {
			// Host ids like US-NY-01: keep full token, only borrow flag/country.
			if strings.HasPrefix(work, iso[1]+"-") {
				if code, name, flag, ok := ResolveCountryToken(iso[1]); ok {
					if out.flag == "" {
						out.flag = flag
					}
					if out.country == "" {
						out.country = name
					}
					if out.code == "" {
						out.code = code
					}
					nameParts = append(nameParts, work)
					continue
				}
			}
			if code, name, flag, ok := ResolveCountryToken(iso[1]); ok {
				if out.flag == "" {
					out.flag = flag
				}
				if out.country == "" {
					out.country = name
				}
				if out.code == "" {
					out.code = code
				}
				left := strings.TrimSpace(strings.Replace(work, iso[0], "", 1))
				if left != "" {
					nameParts = append(nameParts, left)
				}
				continue
			}
		}
		if code, name, flag, ok := ResolveCountryToken(work); ok && len(work) <= 24 {
			if out.flag == "" {
				out.flag = flag
			}
			if out.country == "" {
				out.country = name
			}
			if out.code == "" {
				out.code = code
			}
			continue
		}
		nameParts = append(nameParts, work)
	}
	out.serverName = normSpace(strings.Join(nameParts, " "))
	hostID := out.code != "" && strings.HasPrefix(out.serverName, out.code+"-")
	if out.country != "" && !hostID {
		out.serverName = stripPrefix(out.serverName, out.country)
	}
	if out.code != "" && !hostID {
		out.serverName = stripPrefix(out.serverName, out.code)
	}
	if out.flag != "" {
		out.serverName = stripPrefix(out.serverName, out.flag)
	}
	out.serverName = normSpace(out.serverName)
	if out.flag == "" && out.code != "" {
		out.flag = FlagEmojiForCode(out.code)
	}
	return out
}

func normSpace(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func stripPrefix(rest, token string) string {
	r := strings.TrimSpace(rest)
	if token == "" {
		return r
	}
	if strings.HasPrefix(strings.ToLower(r), strings.ToLower(token)) {
		r = r[len(token):]
		r = strings.TrimLeft(r, " |·•-_–—")
	}
	return strings.TrimSpace(r)
}
