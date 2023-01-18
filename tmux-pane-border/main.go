package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

func getBranchInSync(home, gitRoot, branchName string) string {
	localFile := path.Join(gitRoot, ".git/refs/heads/"+branchName)
	shaHashLocal := readContent(localFile)
	shaHashLocal = strings.ReplaceAll(shaHashLocal, "\n", "")

	section := "branch \"" + branchName + "\""
	// TODO
	gitConfigFile := path.Join(gitRoot, ".git/config")
	cmd := exec.Command(home+"/Documents/python/tools/bin/read_toml_setting", gitConfigFile, "remote", section)
	upstreamBytes, _ := cmd.Output()
	upstream := string(upstreamBytes)
	upstream = strings.ReplaceAll(upstream, "\n", "")

	upstreamFile := path.Join(gitRoot, ".git/refs/remotes/"+upstream+"/"+branchName)
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
	gitRoot = strings.ReplaceAll(gitRoot, "\n", "")

	branchName := getBranchName(gitRoot)
	if len(branchName) > 0 {
		env_vars := os.Environ()
		home := ""
		for _, env_var := range env_vars {
			// fmt.Printf("env_var: %v\n", env_var)
			switch {
			case strings.HasPrefix(env_var, "HOME="):
				home = strings.Split(env_var, "=")[1]
			}
		}

		fmt.Printf("Git %s%s", branchName, getBranchInSync(home, gitRoot, branchName))
	}
}
