//go:build !windows
// +build !windows

package color

import (
	"fmt"
	"io"
	"log"
	"strconv"
)

func getForeground(color Color) string {
	if color == Black {
		return "\033[0m"
	}
	return string(strconv.AppendUint([]byte("\033["), 30+uint64(color), 10)) + "m"
}

func fprintln(w io.Writer, color Color, v string) {
	fmt.Fprintln(w, getForeground(color)+v+getForeground(Black))
}

func fprintlnf(w io.Writer, color Color, format string, v ...any) {
	fprintln(w, color, fmt.Sprintf(format, v...))
}

func println(color Color, v string) {
	fprintln(log.Writer(), color, v)
}

func printlnf(color Color, format string, v ...any) {
	fprintln(log.Writer(), color, fmt.Sprintf(format, v...))
}
