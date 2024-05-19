package main

import (
	"bufio"
	"io/ioutil"
	"os"
)

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, err := os.Open(filename)
	if err != nil {
		debug("file error: %v", err)
		return nil, nil
	}
	reader := bufio.NewReader(file)

	return reader, file
}

func read(filename string) string {
	reader, file := getReader(filename)
	if reader == nil {
		debug("no reader")
		return ""
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		debug("read error: %v", err)
		return ""
	}

	return string(bytes)
}

