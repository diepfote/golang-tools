//go:build debug
// +build debug

package main

import (
	"fmt"
	"os"
)

func debug(message string, arg ...interface{}) {
	msg := fmt.Sprintf(message, arg...)
	fmt.Fprintf(os.Stderr, "[DEBUG]: %v\n", msg)
}
