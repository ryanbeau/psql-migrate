package color

import "io"

// Color is the type of color to be set.
type Color int

// Color for foreground text
const (
	// No change of color
	Black = Color(iota)
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

func Println(color Color, v string) {
	println(color, v)
}

func Printlnf(color Color, format string, v ...any) {
	printlnf(color, format, v...)
}

func Fprintln(w io.Writer, color Color, v string) {
	println(color, v)
}

func Fprintlnf(w io.Writer, color Color, format string, v ...any) {
	printlnf(color, format, v...)
}
