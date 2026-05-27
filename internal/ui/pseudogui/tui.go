package pseudogui

import (
	"fmt"
	"strings"

	"github.com/muhomor/muhomor/internal/ui/console"
	"github.com/muhomor/muhomor/internal/ui/model"
)

const screenWidth = 32

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiCyan   = "\033[36m"
	ansiMuted  = "\033[90m"
	ansiFocus  = "\033[7m"
	ansiTeal   = "\033[36m"
)

// simpleFocus — порядок фокуса как в исходном wireframe (не вертикальное «меню действий»).
type simpleFocus int

const (
	focusConnect simpleFocus = iota
	focusPing
	focusDiagnostics
	focusExport
	focusExtended
)

type simpleView struct {
	conn       model.ConnectionUI
	set        model.SettingsUI
	focus      simpleFocus
	diagOpen   bool
	pingBusy   bool
	sessionSec int
	toast      string
	toastErr   bool
}

// syncSimpleToast mirrors Wails Simple: live errors from conn, optional pin in settings.
func syncSimpleToast(v *simpleView) {
	errText := strings.TrimSpace(v.conn.ErrorText)
	if errText != "" {
		v.toast = errText
		v.toastErr = true
		return
	}
	if !v.set.UIKeepErrorsOnScreen && v.toastErr {
		v.toast = ""
		v.toastErr = false
	}
}

func simpleToastLine(v simpleView) string {
	if strings.TrimSpace(v.conn.ErrorText) != "" {
		return v.conn.ErrorText
	}
	if v.set.UIKeepErrorsOnScreen {
		return v.toast
	}
	if v.toastErr {
		return ""
	}
	return v.toast
}

func drawSimpleScreen(term *console.Terminal, v simpleView, embedded bool) {
	term.ClearScreen()
	fmt.Fprint(term.Out, renderSimpleScreen(v, embedded))
}

func renderSimpleScreen(v simpleView, embedded bool) string {
	var b strings.Builder
	inner := screenWidth - 2

	b.WriteString(boxTop(inner))
	b.WriteString(boxLine(padCenter("muhomor", inner), inner))
	b.WriteString(boxEmpty(inner))

	title := statusTitle(v.conn)
	pill := statusMark(v.conn) + " " + statusPillColor(v.conn) + ansiBold + title + ansiReset
	b.WriteString(boxLine(padCenter(pill, inner), inner))
	act := statusActivity(v.conn)
	if act == "" && uiConnecting(v.conn) {
		act = "Подключение…"
	}
	if act != "" {
		for _, line := range wrapText(act, inner-2) {
			b.WriteString(boxLine(padCenter(ansiDim+truncateRunes(line, inner-2)+ansiReset, inner), inner))
		}
	}
	b.WriteString(boxEmpty(inner))

	cardTitle, caption, flag := serverCardLines(v.conn)
	ping := pingBadge(v.conn, v.pingBusy)
	card1 := fmt.Sprintf("%s  %s%s%s", flag, ansiBold, truncateRunes(cardTitle, 14), ansiReset)
	card1 = padBetween(card1, ping, inner-2)
	b.WriteString(boxLine(" "+highlightCell(cardLineInner(card1, v.focus == focusPing), v.focus == focusPing, inner-2), inner))
	if caption != "" {
		capLine := "   " + ansiDim + caption + ansiReset
		b.WriteString(boxLine(padRight(stripANSI(capLine), inner-2), inner))
	}
	b.WriteString(boxEmpty(inner))

	btn := connectLabel(v.conn)
	for _, row := range powerButtonRows(btn, v.focus == focusConnect) {
		b.WriteString(boxLine(padCenter(row, inner), inner))
	}
	b.WriteString(boxEmpty(inner))

	if v.conn.Connected {
		down := model.FormatBps(v.conn.TrafficDown)
		up := model.FormatBps(v.conn.TrafficUp)
		stats := fmt.Sprintf("↓ %s    ↑ %s", down, up)
		b.WriteString(boxLine(padCenter(stats, inner), inner))
		timer := fmt.Sprintf("⏱ %s", model.FormatDuration(v.sessionSec))
		b.WriteString(boxLine(padCenter(timer, inner), inner))
		b.WriteString(boxEmpty(inner))
	}

	probe := probeBlock(v.conn)
	if probe != "" {
		chev := "▸"
		if v.diagOpen {
			chev = "▾"
		}
		diagLabel := chev + " Диагностика"
		b.WriteString(boxLine(" "+highlightCell(diagLabel, v.focus == focusDiagnostics, inner-2), inner))
		if v.diagOpen {
			for _, line := range wrapText(probe, inner-2) {
				b.WriteString(boxLine(ansiDim+truncateRunes(line, inner-2)+ansiReset, inner))
			}
		}
		b.WriteString(boxEmpty(inner))
	}

	footerLeft := focusablePlain("Экспорт лога", v.focus == focusExport)
	var footerRight string
	if !embedded {
		footerRight = focusablePlain("Расширенный →", v.focus == focusExtended)
	}
	b.WriteString(boxLine(padBetween(footerLeft, footerRight, inner-2), inner))
	b.WriteString(boxBottom(inner))

	if line := simpleToastLine(v); line != "" {
		color := ansiTeal
		if v.toastErr {
			color = ansiRed
		}
		b.WriteString("\n" + color + truncateRunes(line, 60) + ansiReset + "\n")
	}

	hint := "↑↓ — фокус   Enter — действие   q — выход"
	if uiConnecting(v.conn) {
		hint = "Enter — отмена   q — выход"
	}
	b.WriteString(ansiDim + "\n" + hint + ansiReset + "\n")
	return b.String()
}

// highlightCell — DOS-style: инверсия всей ячейки при фокусе.
func highlightCell(text string, focused bool, w int) string {
	plain := padRight(stripANSI(text), w)
	if focused {
		return ansiFocus + plain + ansiReset
	}
	return plain
}

func cardLineInner(s string, focused bool) string {
	if focused {
		return "► " + stripANSI(s)
	}
	return "  " + stripANSI(s)
}

func focusablePlain(label string, focused bool) string {
	if focused {
		return ansiFocus + "► " + label + ansiReset
	}
	return "  " + label
}

func powerButtonRows(label string, focused bool) []string {
	rows := []string{
		"╭──────────╮",
		fmt.Sprintf("│ %-8s │", label),
		"╰──────────╯",
	}
	if focused {
		for i := range rows {
			rows[i] = ansiFocus + rows[i] + ansiReset
		}
	}
	return rows
}

func boxTop(w int) string    { return "┌" + strings.Repeat("─", w) + "┐\n" }
func boxBottom(w int) string { return "└" + strings.Repeat("─", w) + "┘\n" }
func boxSep(w int) string    { return "├" + strings.Repeat("─", w) + "┤\n" }
func boxEmpty(w int) string  { return "│" + strings.Repeat(" ", w) + "│\n" }

func boxLine(s string, w int) string {
	s = trimVisible(s, w)
	if visibleLen(s) < w {
		s += strings.Repeat(" ", w-visibleLen(s))
	}
	return "│" + s + "│\n"
}

func padCenter(s string, w int) string {
	r := []rune(stripANSI(s))
	if len(r) >= w {
		return string(r[:w])
	}
	pad := (w - len(r)) / 2
	return strings.Repeat(" ", pad) + s + strings.Repeat(" ", w-len(r)-pad)
}

func padRight(s string, w int) string {
	r := []rune(stripANSI(s))
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func padBetween(left, right string, w int) string {
	lr := len([]rune(stripANSI(left)))
	rr := len([]rune(stripANSI(right)))
	gap := w - lr - rr
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func trimVisible(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	var out []rune
	visible := 0
	inANSI := false
	for i := 0; i < len(r); i++ {
		ch := r[i]
		if ch == '\033' {
			inANSI = true
			out = append(out, ch)
			continue
		}
		if inANSI {
			out = append(out, ch)
			if ch == 'm' {
				inANSI = false
			}
			continue
		}
		if visible >= max {
			break
		}
		out = append(out, ch)
		visible++
	}
	if inANSI {
		out = append(out, []rune(ansiReset)...)
	}
	return string(out)
}

func visibleLen(s string) int {
	return len([]rune(stripANSI(s)))
}

func stripANSI(s string) string {
	var b strings.Builder
	skip := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			skip = true
			continue
		}
		if skip {
			if s[i] == 'm' {
				skip = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func wrapText(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	var lines []string
	for _, part := range strings.Split(s, "\n") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		r := []rune(part)
		for len(r) > 0 {
			n := width
			if n > len(r) {
				n = len(r)
			}
			lines = append(lines, string(r[:n]))
			r = r[n:]
		}
	}
	return lines
}

func padFrameLines(s string, n int) string {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for len(lines) < n {
		lines = append(lines, "")
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n") + "\n"
}

// menuRow — для подменю (настройки, группы, маршрут): полоса reverse video.
func menuRow(label string, selected, disabled bool, w int) string {
	line := label
	if disabled {
		line = ansiDim + line + ansiReset
	}
	plain := padRight(stripANSI(line), w)
	if selected {
		return ansiFocus + plain + ansiReset
	}
	return " " + plain
}
