package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// 0 ... error only
// 1 ... error, info
// 2 ... error, info, debug
var LogLevel int = 0

func prettyPrintArray(typeOfMessage, nameOfArray string, arr []string) {
	// snatched from https://stackoverflow.com/a/56242100
	s, _ := json.MarshalIndent(arr, "", "\t")
	if typeOfMessage == "INFO" {
		loginfo("%s: %s", typeOfMessage, nameOfArray, string(s))
	} else if typeOfMessage == "DEBUG" {
		logdebug("%s: %s", typeOfMessage, nameOfArray, string(s))
	}
}

func logerr(message string, arg ...interface{}) {
	msg := fmt.Sprintf(message, arg...)
	fmt.Fprintf(os.Stderr, "[ERROR]: %v\n", msg)
}

func loginfo(message string, arg ...interface{}) {
	if LogLevel < 1 {
		return
	}

	msg := fmt.Sprintf(message, arg...)
	fmt.Fprintf(os.Stderr, "[INFO]: %v\n", msg)
}

func logdebug(message string, arg ...interface{}) {
	if LogLevel < 2 {
		return
	}

	msg := fmt.Sprintf(message, arg...)
	fmt.Fprintf(os.Stderr, "[DEBUG]: %v\n", msg)
}
