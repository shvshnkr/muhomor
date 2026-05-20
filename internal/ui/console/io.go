package console

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/muhomor/muhomor/internal/appcore"
)

// IO is stdin/stdout terminal I/O (Linux/Windows desktop).
type IO struct {
	In  io.Reader
	Out io.Writer
}

func StdIO() *IO {
	return &IO{In: os.Stdin, Out: os.Stdout}
}

func (c *IO) Line(text string) {
	fmt.Fprintln(c.Out, text)
}

func (c *IO) Block(text string) {
	fmt.Fprintln(c.Out, text)
}

// ReadLine reads one line with optional prompt on Out.
func (c *IO) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(c.Out, prompt)
	}
	sc := bufio.NewScanner(c.In)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(sc.Text()), nil
}

var _ appcore.OutputSink = (*IO)(nil)
