package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type RsyncInfo struct {
	RemoteIP string
	SshUser  string
	SshKey   string
}

func doSync(fileToSync string, RsyncInfo rsyncInfo) {
	fmt.Printf("[INFO]: syncing: %v\n", fileToSync)

	downloadUrl := getDownloadUrl(fileToSync)
	if len(downloadUrl) == 0 {
		fmt.Printf("[WARNING]: downloadUrl empty. Not syncing!\n")
		return
	}

	directoryToSyncTo := filepath.Dir(fileToSync)
	fileToSync = filepath.Base(fileToSync)
	fmt.Printf("[INFO]: syncing to DIR: %v\n", directoryToSyncTo)

	// Create dir if it does not exist
	err := os.MkdirAll(directoryToSyncTo, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Mkdir: %v\n", err)
	}

	var Command cmd = nil
	if rsyncInfo {
		cmd = exec.Command("echo", "rsync", "--dry-run", "--delete", "-av", "--exclude", ".DS_Store", "--exclude", ".localized", "--exclude", "no-sync/", "-e", "'ssh -i "+rsyncInfo.SshKey+"'", "'"+fileToSync+"'", rsyncInfo.SshUser+"@"+rsyncInfo.RemoteIP+":'"+fileToSync+"'")
	} else {
		cmd = exec.Command("youtube-dl", "--add-metadata", "-i", "-f", "22", downloadUrl)
	}
	cmd.Dir = directoryToSyncTo

	var stdErrBuffer, stdOutBuffer bytes.Buffer
	multiWriterStdout := io.MultiWriter(os.Stdout, &stdOutBuffer)
	multiWriterStdErr := io.MultiWriter(os.Stderr, &stdErrBuffer)

	cmd.Stdout = multiWriterStdout
	cmd.Stderr = multiWriterStdErr

	// Execute the command & wait for it to exit
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] youtube-dl: %v\n", err)
	}
	// Stream stdout & stderr to parent process stdout & stderr
	fmt.Printf("%v\n", stdOutBuffer.String())
	fmt.Fprintf(os.Stderr, "%v\n", stdErrBuffer.String())

	cmd.Run()
}

func getDownloadUrl(fileToSync string) string {
	// re := regexp.MustCompile(`\r?\n`)

	// don't forget this matches a reversed youtube id
	// e.g.:
	// 4pm.]QXqBuJpErb6[ SGT - sgnihT eciN evaH tnaC eW yhW sI sihT _ SMLAER ELTTAB/sevitcepsorteR - sgniht ecin evah tnac ew yhw si sihT/tiucsiblatot
	re := regexp.MustCompile(`^[A-z0-9]{2,6}\.\]*([A-z0-9-]{11})\[*`)
	regexSubmatches := re.FindStringSubmatch(reverse(fileToSync))

	if len(regexSubmatches) < 2 {
		return ""
	}
	// fmt.Printf("[DEBUG]: regexSubmatches %#v\n", regexSubmatches)
	youtubeId := reverse(regexSubmatches[1])
	// fmt.Printf("[DEBUG]: youtubeId: %v\n", youtubeId)
	downloadUrl := ""
	if len(youtubeId) > 0 {
		downloadUrl = "https://youtu.be/" + youtubeId
	}
	fmt.Printf("[INFO]: url is: %v\n", downloadUrl)

	return downloadUrl
}

func reverse(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func yesNoWrapperDelete() func(question string) bool {
	scanner := bufio.NewScanner(os.Stdin)
	return func(question string) bool {
		fmt.Printf("Would you like to remove '%v' [y|N]?\n", question)

		// reads user input until \n by default
		scanner.Scan()

		// Holds the string that was scanned
		text := scanner.Text()
		if text == "y" || text == "Y" {
			return true
		} else {
			return false
		}
	}
}
func yesNoWrapperDownload() func(question string) bool {
	scanner := bufio.NewScanner(os.Stdin)
	return func(question string) bool {
		fmt.Printf("Would you like to download '%v'? [Y|n]?\n", question)

		// reads user input until \n by default
		scanner.Scan()

		// Holds the string that was scanned
		text := scanner.Text()
		if text == "n" || text == "N" {
			return false
		} else {
			return true
		}
	}
}
func yesNoWrapperAskOnEachDownload() func() bool {
	scanner := bufio.NewScanner(os.Stdin)
	return func() bool {
		fmt.Printf("Would you like to approve every download? [y|N]?\n")

		// reads user input until \n by default
		scanner.Scan()

		// Holds the string that was scanned
		text := scanner.Text()
		if text == "y" || text == "Y" {
			return true
		} else {
			return false
		}
	}
}

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] file error: %v\n", err)
		return nil, nil
	}
	reader := bufio.NewReader(file)

	return reader, file
}

func read(filename string) string {
	reader, file := getReader(filename)
	if reader == nil {
		fmt.Fprintf(os.Stderr, "[ERROR] no reader\n")
		return ""
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] read error: %v\n", err)
		return ""
	}

	return string(bytes)
}

func walkPath(dirname string, excludedDirs, filesToSync []string, read_only bool, yesNo func(string) bool) ([]string, error) {
	var filesVisited []string

	// prettyPrintArray("DEBUG", "excludedDirs", excludedDirs)

	err := filepath.Walk(dirname, func(_path string, fileinfo os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] prevent panic by handling failure accessing a path %q: %v\n", _path, err)
			return err
		}

		if fileinfo.IsDir() && stringInArray(excludedDirs, _path) {
			// fmt.Printf("[DEBUG] skipping excluded path: %v\n", _path)
			return filepath.SkipDir
		}
		// } else {
		// 	fmt.Printf("[DEBUG] not skipping path: %v\n", _path)
		// }

		if !fileinfo.IsDir() {
			// fmt.Printf("[DEBUG] _path not dir %v\n", _path)
			filesVisited = append(filesVisited, _path)

			if !stringInArrayCheckForIntegerPrefixes(filesToSync, _path) && !read_only {
				if strings.Contains(_path, ".DS_Store") {
					return nil
				}
				// answer := yesNo(_path)
				answer := false
				if answer {
					fmt.Printf("[INFO] removing: %v\n", _path)
					err := os.Remove(_path)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[ERROR]: %v\n", err)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		// log error in upper block
		return nil, err
	}

	return filesVisited, err
}

func prettyPrintArray(typeOfMessage, nameOfArray string, arr []string) {
	// snatched from https://stackoverflow.com/a/56242100
	s, _ := json.MarshalIndent(arr, "", "\t")
	fmt.Printf("[%s]: %s: %s\n", typeOfMessage, nameOfArray, string(s))
}

func arrayInString(arr []string, str string) bool {
	// fmt.Printf("[DEBUG] arrayInString arr: %#v\n", arr)
	// fmt.Printf("[DEBUG] arrayInString str: %v\n", str)
	for _, a := range arr {

		//	func HasPrefix(s, prefix string) bool
		//		HasPrefix tests whether the string s begins with prefix.
		if strings.HasPrefix(a, str) {
			return true
		}
	}
	return false
}

func stringInArrayCheckForIntegerPrefixes(arr []string, str string) bool {
	// fmt.Printf("[DEBUG] stringInArrayCheckForIntegerPrefixes arr: %#v\n", arr)
	// fmt.Printf("[DEBUG] stringInArraysCheckForIntegerPrefixes str: %v\n", str)

	// DEBUG
	// 	fmt.Printf("[DEBUG] ARRRAY: %#v\n", arr)
	// 	os.Exit(0)

	for _, element := range arr {

		// fmt.Printf("[DEBUG] element: %v = str: %v?\n", element, str)

		//	func HasPrefix(s, prefix string) bool
		//		HasPrefix tests whether the string s begins with prefix.
		if strings.HasPrefix(str, element) {
			return true
		}

		// TODO use full path in check

		strWithoutPath := reverse(strings.Split(reverse(str), "/")[0])
		re := regexp.MustCompile(`^0?[0-9]{1,6} `) // filename starts with 01 or 11 etc.
		strWithoutPathMatches := re.FindStringSubmatch(strWithoutPath)

		tail := ""
		if len(strWithoutPathMatches) > 0 {
			// remove prepended integer
			tail = strings.SplitN(strWithoutPath, " ", 2)[1]
		} else {
			tail = strWithoutPath
		}

		elementWithoutPath := reverse(strings.Split(reverse(element), "/")[0])
		elementWithoutPathMatches := re.FindStringSubmatch(elementWithoutPath)
		// fmt.Printf("[DEBUG] elementWithoutPath: %#v\n", elementWithoutPath)
		// fmt.Printf("[DEBUG] elementWithoutPathMatches: %#v\n", elementWithoutPathMatches)

		elementTail := ""
		if len(elementWithoutPathMatches) > 0 {
			// remove prepended integer
			elementTail = strings.SplitN(elementWithoutPath, " ", 2)[1]
		} else {
			elementTail = elementWithoutPath
		}

		if strings.HasPrefix(tail, elementTail) {
			// fmt.Printf("[DEBUG]\t\ttail: %v\n", tail)
			// fmt.Printf("[DEBUG]  elementTail: %v\n", elementTail)
			// fmt.Printf("\n")
			return true
		}

	}
	return false
}

func stringInArray(arr []string, str string) bool {
	// fmt.Printf("[DEBUG] stringInArray arr: %#v\n", arr)
	// fmt.Printf("[DEBUG] stringInArray str: %v\n", str)
	for _, a := range arr {

		//	func HasPrefix(s, prefix string) bool
		if strings.HasPrefix(str, a) || strings.HasSuffix(str, a) {
			return true
		}
	}
	return false
}

// Nicked from https://siongui.github.io/2018/03/14/go-set-difference-of-two-arrays/
func getArrayDiff(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			if len(getDownloadUrl(item)) <= 0 {
				fmt.Printf("[INFO] Will not ask if `%v` should be downloaded (no youtube id)\n", item)
				continue
			}
			diff = append(diff, item)
		}
	}
	return
}

func main() {

	env_vars := os.Environ()
	home := ""
	remoteIP := ""
	sshUser := ""
	sshKey := ""
	for _, env_var := range env_vars {
		switch {
		case strings.HasPrefix(env_var, "HOME="):
			home = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_REMOTE_ADDRESS="):
			remoteIP = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_USER="):
			sshUser = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_KEY="):
			sshKey = strings.Split(env_var, "=")[1]
		}
	}

	var RsyncInfo rsyncInfo = nil
	if len(sshUser) > 0 {
		rsyncInfo = RsyncInfo{
			RemoteIP: remoteIP,
			SshUser:  sshUser,
			SshKey:   sshKey,
		}
	}

	os.Chdir(os.Args[1])
	read_only := false

	if len(os.Args) > 2 && os.Args[2] == "report-files" {
		read_only = true
	}

	tmpExcludedDirs := strings.Split(read(path.Join(home, "Documents/config/video-syncer-excluded-dirs.conf")), "\n")
	// remove empty string = last element
	tmpExcludedDirs = tmpExcludedDirs[:len(tmpExcludedDirs)-1]

	var excludedDirs []string = nil
	for _, exclude := range tmpExcludedDirs {
		if exclude != "" {
			excludedDirs = append(excludedDirs, exclude)
		}
	}

	syncFileContentsLinux := read(path.Join(home, "Documents/misc/videos", "videos-home.txt"))
	syncFileContentsDarwin := read(path.Join(home, "Documents/misc/videos", "videos-work.txt"))
	// strip mpv commands
	syncFileContentsLinux = strings.Split(syncFileContentsLinux, "\n\n")[0]
	// strip mpv commands
	syncFileContentsDarwin = strings.Split(syncFileContentsDarwin, "\n\n")[0]

	filesToSyncLinux := strings.Split(syncFileContentsLinux, "\n")[1:]
	filesToSyncDarwin := strings.Split(syncFileContentsDarwin, "\n")[1:]

	var filesToSync []string = nil
	if runtime.GOOS != "linux" {
		// if linux use the darwin sync contents
		// and vice-versa
		filesToSync = filesToSyncLinux
	} else {
		filesToSync = filesToSyncDarwin
	}
	// fmt.Printf("[DEBUG]: GOOS: %#v\n", runtime.GOOS)
	// prettyPrintArray("DEBUG", "filesToSync", filesToSync)

	yesNoDelete := yesNoWrapperDelete()
	filesVisited, err := walkPath(".", excludedDirs, filesToSync, read_only, yesNoDelete)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] walkPath error: %v\n", err)
	}

	if read_only {
		for _, fileVisited := range filesVisited {
			if fileVisited == ".DS_Store" {
				continue
			}
			fmt.Printf("%v\n", fileVisited)
		}
		return
	}

	var filesToDownload []string = nil
	if runtime.GOOS != "linux" {
		// if linux use the darwin sync contents
		// and vice-versa
		filesToDownload = getArrayDiff(filesToSyncLinux, filesToSyncDarwin)
	} else {
		filesToDownload = getArrayDiff(filesToSyncDarwin, filesToSyncLinux)
	}
	prettyPrintArray("INFO", "filesToDownload", filesToDownload)

	yesNoAskOnEachDownload := yesNoWrapperAskOnEachDownload()
	yesNoDownload := yesNoWrapperDownload()

	askOnEachDownload := yesNoAskOnEachDownload()

	for _, fileToDownload := range filesToDownload {
		if stringInArray(filesVisited, fileToDownload) {
			// fmt.Printf("[DEBUG]: file seen, not syncing: %v\n", fileToSync)
			continue
		}
		info, err := os.Stat(fileToDownload)
		if err != nil {
			// fmt.Printf("[DEBUG]: %v\n", err)

			// Do not skip, if the file does not exist
			// we want to sync it.
		} else {
			// Skip directories

			if info.IsDir() {
				// fmt.Printf("[DEBUG] not syncing '%v'. This is a directory.\n", fileToSync)
				continue
			} else {
				// we might want to continue a snyc --> fall through
			}
		}

		if stringInArray(excludedDirs, fileToDownload) {
			// fmt.Printf("[DEBUG] skipping excluded path: %v\n", fileToSync)
			continue
		}

		if askOnEachDownload {
			if yesNoDownload(fileToDownload) {
				doSync(fileToDownload, rsyncInfo)
			}
		} else {
			doSync(fileToDownload, rsyncInfo)
		}
	}
}
