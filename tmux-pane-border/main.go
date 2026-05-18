package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func getBranchName(gitRoot string) string {
	fileToRead := path.Join(gitRoot, ".git/HEAD")
	branchInfo := read(fileToRead)

	branchInfoSplit := strings.Split(branchInfo, "/")[2:]
	branchInfo = strings.Join(branchInfoSplit, "/")

	return strings.ReplaceAll(branchInfo, "\n", "")
}

func getBranchInSync(home, gitRoot, branchName string) string {
	localFile := path.Join(gitRoot, ".git/refs/heads/"+branchName)
	shaHashLocal := read(localFile)
	shaHashLocal = strings.ReplaceAll(shaHashLocal, "\n", "")

	section := "branch \"" + branchName + "\""
	// TODO
	gitConfigFile := path.Join(gitRoot, ".git/config")
	cmd := exec.Command("read-ini-setting", gitConfigFile, "remote", section)
	upstreamBytes, _ := cmd.Output()
	upstream := string(upstreamBytes)
	upstream = strings.ReplaceAll(upstream, "\n", "")

	upstreamFile := path.Join(gitRoot, ".git/refs/remotes/"+upstream+"/"+branchName)
	shaHashUpstream := read(upstreamFile)
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
		home := os.Getenv("HOME")

		fmt.Printf("Git %s%s", branchName, getBranchInSync(home, gitRoot, branchName))
	}
}
