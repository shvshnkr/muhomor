package console

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Key is a normalized terminal key for TUI menus.
type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyQuit
	// KeyDigit0..KeyDigit9 — press number to jump in list pickers.
	KeyDigit0 Key = 40 + iota
	KeyDigit1
	KeyDigit2
	KeyDigit3
	KeyDigit4
	KeyDigit5
	KeyDigit6
	KeyDigit7
	KeyDigit8
	KeyDigit9
)

// Terminal provides raw-mode stdin reads and screen control.
type Terminal struct {
	In  *os.File
	Out io.Writer
	restore func()
}

// NewTerminal wraps stdin/stdout for interactive TUI.
func NewTerminal(io *IO) *Terminal {
	in := os.Stdin
	if f, ok := io.In.(*os.File); ok {
		in = f
	}
	out := io.Out
	if out == nil {
		out = os.Stdout
	}
	return &Terminal{In: in, Out: out}
}

// EnterRaw switches stdin to raw mode. Call LeaveRaw before exit.
func (t *Terminal) EnterRaw() error {
	if !term.IsTerminal(int(t.In.Fd())) {
		return fmt.Errorf("stdin is not a terminal")
	}
	state, err := term.MakeRaw(int(t.In.Fd()))
	if err != nil {
		return err
	}
	t.restore = func() { _ = term.Restore(int(t.In.Fd()), state) }
	fmt.Fprint(t.Out, "\033[?25l") // hide cursor
	return nil
}

// LeaveRaw restores canonical mode and shows the cursor.
func (t *Terminal) LeaveRaw() {
	fmt.Fprint(t.Out, "\033[?25h")
	if t.restore != nil {
		t.restore()
		t.restore = nil
	}
}

// ClearScreen erases the viewport and moves the cursor home.
func (t *Terminal) ClearScreen() {
	fmt.Fprint(t.Out, "\033[H\033[2J")
}

// DrawHome reprints at cursor home without clearing (fixed-height frames avoid ghost lines).
func (t *Terminal) DrawHome(s string) {
	fmt.Fprint(t.Out, "\033[H", s)
}

// UseAltScreen switches to the terminal alternate screen buffer.
func (t *Terminal) UseAltScreen() {
	fmt.Fprint(t.Out, "\033[?1049h\033[H\033[2J")
}

// RestoreAltScreen returns to the primary screen buffer.
func (t *Terminal) RestoreAltScreen() {
	fmt.Fprint(t.Out, "\033[?1049l")
}

// ReadKey blocks until a navigation key is read.
func (t *Terminal) ReadKey() (Key, error) {
	buf := make([]byte, 32)
	for {
		n, err := t.In.Read(buf)
		if err != nil {
			return KeyNone, err
		}
		if n == 0 {
			continue
		}
		b := buf[:n]
		if k := decodeKey(b); k != KeyNone {
			return k, nil
		}
	}
}

func decodeKey(b []byte) Key {
	if len(b) == 1 {
		switch b[0] {
		case 3: // Ctrl+C
			return KeyQuit
		case 13, 10:
			return KeyEnter
		case 27:
			return KeyEsc
		case 'q', 'Q':
			return KeyQuit
		case '0':
			return KeyDigit0
		case '1':
			return KeyDigit1
		case '2':
			return KeyDigit2
		case '3':
			return KeyDigit3
		case '4':
			return KeyDigit4
		case '5':
			return KeyDigit5
		case '6':
			return KeyDigit6
		case '7':
			return KeyDigit7
		case '8':
			return KeyDigit8
		case '9':
			return KeyDigit9
		}
		return KeyNone
	}
	if len(b) >= 3 && b[0] == 27 && b[1] == '[' {
		switch b[2] {
		case 'A':
			return KeyUp
		case 'B':
			return KeyDown
		case 'C':
			return KeyRight
		case 'D':
			return KeyLeft
		}
	}
	// Windows arrow keys: ESC O A
	if len(b) >= 3 && b[0] == 27 && b[1] == 'O' {
		switch b[2] {
		case 'A':
			return KeyUp
		case 'B':
			return KeyDown
		case 'C':
			return KeyRight
		case 'D':
			return KeyLeft
		}
	}
	return KeyNone
}
