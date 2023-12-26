//go:build debug
// +build debug

package main

import (
	"fmt"
	"os"
)

// 0 ... error only
// 1 ... error, info
// 2 ... error, info, debug
var LogLevel int = 2

func debug(message string, arg ...interface{}) {
	msg := fmt.Sprintf(message, arg...)
	fmt.Fprintf(os.Stderr, "[DEBUG]: %v\n", msg)
}
