//go:build !debug
// +build !debug

package main

// 0 ... error only
// 1 ... error, info
// 2 ... error, info, debug
var LogLevel int = 0

func debug(message string, arg ...interface{}) {
}
