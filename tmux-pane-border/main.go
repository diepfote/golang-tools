package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, _ := os.Open(filename)
	// file, error := os.Open(filename)
	// if error != nil {
	// 	fmt.Printf("file error: %v", error)
	// }
	reader := bufio.NewReader(file)

	return reader, file
}

func readContent(filename string) string {
	reader, file := getReader(filename)
	defer file.Close()

	bytes, _ := ioutil.ReadAll(reader)
	// bytes, error := ioutil.ReadAll(reader)
	// if error != nil {
	// 	fmt.Printf("read error: %v", error)
	// }

	return string(bytes)
}

func main() {

	cwd := os.Args[1]

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	// cwd := strings.TrimSuffix(string(bytes.Trim(buffer, "\x00")), "\n")
	cmd.Dir = cwd
	gitRootBytes, _ := cmd.Output()
	gitRoot := string(gitRootBytes)

	re := regexp.MustCompile(`\r?\n`)
	gitRoot = re.ReplaceAllString(gitRoot, "")

	fileToRead := path.Join(gitRoot, ".git/HEAD")
	branchInfo := readContent(fileToRead)

	branchInfoSplit := strings.Split(branchInfo, "/")[2:]
	branchInfo = strings.Join(branchInfoSplit, "/")

	if len(branchInfo) > 0 {
		fmt.Printf("Git %s", strings.ReplaceAll(branchInfo, "\n", ""))
	}
}
