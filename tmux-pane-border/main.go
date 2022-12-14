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

func getBranchName(gitRoot string) string {
	fileToRead := path.Join(gitRoot, ".git/HEAD")
	branchInfo := readContent(fileToRead)

	branchInfoSplit := strings.Split(branchInfo, "/")[2:]
	branchInfo = strings.Join(branchInfoSplit, "/")

	return strings.ReplaceAll(branchInfo, "\n", "")
}

func getBranchInSync(gitRoot, branchName string) string {
	localFile := path.Join(gitRoot, ".git/refs/heads/"+branchName)
	shaHashLocal := readContent(localFile)
	shaHashLocal = strings.ReplaceAll(shaHashLocal, "\n", "")

	upstreamFile := path.Join(gitRoot, ".git/refs/remotes/origin/"+branchName)
	shaHashUpstream := readContent(upstreamFile)
	shaHashUpstream = strings.ReplaceAll(shaHashUpstream, "\n", "")

	if len(shaHashUpstream) < 1 {
		return " -no-upstream-"
	} else if shaHashLocal != shaHashUpstream {
		return " -diverges-"
	}

	return ""
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

	branchName := getBranchName(gitRoot)
	if len(branchName) > 0 {
		fmt.Printf("Git %s%s", branchName, getBranchInSync(gitRoot, branchName))
	}
}
