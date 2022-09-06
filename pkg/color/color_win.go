//go:build windows
// +build windows

// Windows text color from https://github.com/daviddengcn/go-colortext

package color

import (
	"fmt"
	"io"
	"log"
	"syscall"
	"unsafe"
)

const (
	blue      = uint16(0x0001)
	green     = uint16(0x0002)
	red       = uint16(0x0004)
	intensity = uint16(0x0008)

	mask = blue | green | red | intensity
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procSetConsoleTextAttribute    = kernel32.NewProc("SetConsoleTextAttribute")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")

	hStdout        uintptr
	initScreenInfo *consoleScreenBufferInfo

	colors = []uint16{
		0,
		red,
		green,
		red | green,
		blue,
		red | blue,
		green | blue,
		red | green | blue,
	}
)

func init() {
	const stdOutputHandle = uint32(-11 & 0xFFFFFFFF)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle = kernel32.NewProc("GetStdHandle")
	hStdout, _, _ = procGetStdHandle.Call(uintptr(stdOutputHandle))
	initScreenInfo = getConsoleScreenBufferInfo(hStdout)
	syscall.LoadDLL("")
}

type coord struct {
	X, Y int16
}

type rect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	DwSize              coord
	DwCursorPosition    coord
	WAttributes         uint16
	SrWindow            rect
	DwMaximumWindowSize coord
}

func getConsoleScreenBufferInfo(hConsoleOutput uintptr) *consoleScreenBufferInfo {
	var csbi consoleScreenBufferInfo
	if ret, _, _ := procGetConsoleScreenBufferInfo.Call(hConsoleOutput, uintptr(unsafe.Pointer(&csbi))); ret == 0 {
		return nil
	}
	return &csbi
}

func setConsoleTextAttribute(hConsoleOutput uintptr, wAttributes uint16) bool {
	ret, _, _ := procSetConsoleTextAttribute.Call(hConsoleOutput, uintptr(wAttributes))
	return ret != 0
}

func resetColor() {
	if initScreenInfo == nil { // No console info - Ex: stdout redirection
		return
	}
	setConsoleTextAttribute(hStdout, initScreenInfo.WAttributes)
}

func setColor(fg Color) {
	if fg == Black {
		cbufinfo := getConsoleScreenBufferInfo(hStdout)
		if cbufinfo == nil { // No console info - Ex: stdout redirection
			return
		}
		setConsoleTextAttribute(hStdout, cbufinfo.WAttributes)
	} else {
		setConsoleTextAttribute(hStdout, 0&^mask|colors[fg])
	}
}

func fprintln(w io.Writer, color Color, v string) {
	setColor(color)
	fmt.Fprintln(w, v)
	resetColor()
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
