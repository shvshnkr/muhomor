package pseudogui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/muhomor/muhomor/internal/ui/console"
)

const pickerWidth = 56

// RunListPicker shows a DOS-style list (↑↓, цифры, Enter). Returns index and ok=false on cancel.
func RunListPicker(ctx context.Context, io *console.IO, title string, options []string, selected int) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}
	if selected < 0 || selected >= len(options) {
		selected = 0
	}

	term := console.NewTerminal(io)
	if err := term.EnterRaw(); err != nil {
		return pickListFallback(io, title, options, selected)
	}
	defer term.LeaveRaw()

	keys := make(chan console.Key, 8)
	errc := make(chan error, 1)
	go func() {
		for {
			k, err := term.ReadKey()
			if err != nil {
				errc <- err
				return
			}
			keys <- k
		}
	}()

	sel := selected
	lastFrame := ""
	paint := func() {
		frame := renderPickList(title, options, sel)
		if frame == lastFrame {
			return
		}
		if lastFrame == "" {
			term.ClearScreen()
		} else {
			term.DrawHome(frame)
		}
		fmt.Fprint(term.Out, frame)
		lastFrame = frame
	}
	paint()

	for {
		select {
		case <-ctx.Done():
			return sel, false
		case <-errc:
			return sel, false
		case k := <-keys:
			switch k {
			case console.KeyQuit, console.KeyEsc:
				return sel, false
			case console.KeyUp:
				if sel <= 0 {
					sel = len(options) - 1
				} else {
					sel--
				}
				paint()
			case console.KeyDown:
				if sel >= len(options)-1 {
					sel = 0
				} else {
					sel++
				}
				paint()
			case console.KeyEnter:
				return sel, true
			default:
				if idx, ok := keyToIndex(k); ok && idx < len(options) {
					sel = idx
					paint()
				}
			}
		}
	}
}

func keyToIndex(k console.Key) (int, bool) {
	if k < console.KeyDigit0 || k > console.KeyDigit9 {
		return 0, false
	}
	return int(k - console.KeyDigit0), true
}

func renderPickList(title string, options []string, selected int) string {
	inner := pickerWidth - 2
	var b strings.Builder
	b.WriteString(boxTop(inner))
	b.WriteString(boxLine(padCenter(ansiBold+title+ansiReset, inner), inner))
	b.WriteString(boxSep(inner))
	for i, opt := range options {
		label := pickerOptionLabel(i, opt)
		line := menuRow(label, i == selected, false, inner-2)
		b.WriteString(boxLine(line, inner))
	}
	b.WriteString(boxBottom(inner))
	b.WriteString(ansiDim + "\n↑↓ — выбор   0-9 — пункт   Enter — применить   Esc — отмена" + ansiReset + "\n")
	// fixed height: header 3 + options + footer
	lines := 3 + len(options) + 2
	return padFrameLines(b.String(), lines)
}

func pickerOptionLabel(i int, opt string) string {
	if strings.HasPrefix(opt, "[") {
		return opt
	}
	return fmt.Sprintf("%d  %s", i, opt)
}

func pickListFallback(io *console.IO, title string, options []string, selected int) (int, bool) {
	io.Line("")
	io.Line("--- " + title + " ---")
	for i, opt := range options {
		mark := " "
		if i == selected {
			mark = "*"
		}
		io.Line(fmt.Sprintf(" %s [%d] %s", mark, i, opt))
	}
	raw, err := io.ReadLine("Номер (Enter — отмена): ")
	if err != nil || strings.TrimSpace(raw) == "" {
		return selected, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n >= len(options) {
		io.Line("Неверный номер")
		return selected, false
	}
	return n, true
}
