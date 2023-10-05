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

// 0 ... error only
// 1 ... error, info
// 2 ... error, info, debug
var LogLevel int = 0
var ReadOnly bool = false

type RsyncInfo struct {
	RemoteIP string
	SshUser  string
	SshKey   string
}

type DirectoryInfo struct {
	LocalVideoDirectory  string
	RemoteVideoDirectory string
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

func doDownload(fileToDownload, home string, directoryInfo *DirectoryInfo, rsyncInfoPtr *RsyncInfo) {
	loginfo("downloading: %v", fileToDownload)

	downloadUrl := getDownloadUrl(fileToDownload)
	if len(downloadUrl) == 0 {
		fmt.Fprintf(os.Stderr, "[WARNING]: downloadUrl empty. Not downloading!\n")
		return
	}

	directoryToSyncTo := directoryInfo.LocalVideoDirectory + "/" + filepath.Dir(fileToDownload)
	fileBase := filepath.Base(fileToDownload)
	loginfo("downloading to DIR: %v", directoryToSyncTo)

	// Create dir if it does not exist
	err := os.MkdirAll(directoryToSyncTo, 0755)
	if err != nil {
		logerr("Mkdir: %v", err)
	}

	cmd := exec.Command("youtube-dl", "--add-metadata", "-i", "-f", "22", downloadUrl)
	if rsyncInfoPtr != nil {
		cmd = exec.Command(home+"/Documents/scripts/video-syncer-rsync-helper.sh", rsyncInfoPtr.SshKey, rsyncInfoPtr.SshUser+"@"+rsyncInfoPtr.RemoteIP+":"+directoryInfo.RemoteVideoDirectory+"/"+fileToDownload, directoryInfo.LocalVideoDirectory+"/"+directoryToSyncTo+"/"+fileBase)
	} else {
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
		if rsyncInfoPtr != nil {
			logerr("rsync: %v", err)
		} else {
			logerr("yt-dlp: %v", err)
		}
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
	// logdebug("regexSubmatches %#v", regexSubmatches)
	youtubeId := reverse(regexSubmatches[1])
	// logdebug("youtubeId: %v", youtubeId)
	downloadUrl := ""
	if len(youtubeId) > 0 {
		downloadUrl = "https://youtu.be/" + youtubeId
	}
	logdebug("url is: %v", downloadUrl)

	return downloadUrl
}

func reverse(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func yesNo(question string) bool {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%v [y|N]?\n", question)

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

func getReader(filename string) (*bufio.Reader, *os.File) {
	file, err := os.Open(filename)
	if err != nil {
		logerr("file error: %v", err)
		return nil, nil
	}
	reader := bufio.NewReader(file)

	return reader, file
}

func read(filename string) string {
	reader, file := getReader(filename)
	if reader == nil {
		logerr("no reader")
		return ""
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		logerr("read error: %v", err)
		return ""
	}

	return string(bytes)
}

func walkPath(localVideoDirName string, excludedDirs, excludedFilenames, filesToSync []string, askAboutDeletions bool, yesNo func(string) bool) ([]string, error) {
	var filesVisited []string

	logdebug("walk from %v", localVideoDirName)
	prettyPrintArray("DEBUG", "excludedDirs", excludedDirs)

	err := filepath.Walk(localVideoDirName, func(_path string, fileinfo os.FileInfo, err error) error {
		if err != nil {
			logerr("prevent panic by handling failure accessing a path %q: %v", _path, err)
			return err
		}

		if fileinfo.IsDir() && stringInArray(excludedDirs, _path) {
			logdebug("skipping excluded path: %v", _path)
			return filepath.SkipDir
		} else if fileinfo.IsDir() {
			logdebug("skipping directory (but we will look into its files): %v", _path)
			return nil
		} else if stringInArray(excludedFilenames, filepath.Base(_path)) {
			logdebug("skipping excluded filename: %v", _path)
			return nil
		}
		// else {
		// 	logdebug("not skipping path: %v", _path)
		// }

		pathWithoutLocalVideoDir := strings.Split(_path, localVideoDirName+"/")[1]
		filesVisited = append(filesVisited, pathWithoutLocalVideoDir)

		if !stringInArrayCheckForIntegerPrefixes(filesToSync, _path) && !ReadOnly {
			if askAboutDeletions {
				//
				// TODO allow to skip entire directories
				//
				answer := yesNo("Would you like to remove '" + _path + "'")

				if answer {
					loginfo("removing: %v", _path)
					err := os.Remove(_path)
					if err != nil {
						logerr("%v", err)
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
	if typeOfMessage == "INFO" {
		loginfo("%s: %s", typeOfMessage, nameOfArray, string(s))
	} else if typeOfMessage == "DEBUG" {
		logdebug("%s: %s", typeOfMessage, nameOfArray, string(s))
	}
}

func arrayInString(arr []string, str string) bool {
	// prettyPrintArray("DEBUG", "arr", arr)
	// logdebug("arrayInString str: %v", str)
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
	// prettyPrintArray("DEBUG", "stringInArrayCheckForIntegerPrefixes", arr)
	// logdebug("stringInArraysCheckForIntegerPrefixes str: %v", str)

	for _, element := range arr {

		// logdebug("element: %v = str: %v?", element, str)

		//	func HasPrefix(s, prefix string) bool
		//		HasPrefix tests whether the string s begins with prefix.
		if strings.HasPrefix(str, element) {
			return true
		}

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
		// logdebug("elementWithoutPath: %#v", elementWithoutPath)
		// logdebug("elementWithoutPathMatches: %#v", elementWithoutPathMatches)

		elementTail := ""
		if len(elementWithoutPathMatches) > 0 {
			// remove prepended integer
			elementTail = strings.SplitN(elementWithoutPath, " ", 2)[1]
		} else {
			elementTail = elementWithoutPath
		}

		if strings.HasPrefix(tail, elementTail) {
			// logdebug("\t\ttail: %v", tail)
			// logdebug("\telementTail: %v", elementTail)
			return true
		}

	}
	return false
}

func stringInArray(arr []string, str string) bool {
	// prettyPrintArray("DEBUG", "arr", arr)
	// logdebug("stringInArray str: %v", str)
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
				//
				// change behavior if we use rsync
				//
				loginfo("Will not ask if `%v` should be downloaded (no youtube id)", item)
				continue
			}
			diff = append(diff, item)
		}
	}
	return
}

func cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames []string, approveEveryDownload bool) (filteredFiles []string) {
	//
	// TODO cleanup: for loop is ugly. should this logic not live elsewhere?
	//               or is this not already taken care of in "reporting"
	//

	for _, fileToDownload := range filesToDownload {
		if stringInArray(excludedFilenames, fileToDownload) {
			logdebug("filename excluded, not syncing: %v", fileToDownload)
			continue
		} else if stringInArray(filesVisited, fileToDownload) {
			logdebug("file seen, not syncing: %v", fileToDownload)
			continue
		}
		info, err := os.Stat(fileToDownload)
		if err != nil {
			logdebug("%v", err)

			// Do not skip, if the file does not exist
			// we want to sync it.
		} else {
			// Skip directories

			if info.IsDir() {
				logdebug("not syncing '%v'. This is a directory.", fileToDownload)
				continue
			}
			// else {
			// 	// we might want to continue a sync --> fall through
			// }
		}

		if stringInArray(excludedDirs, fileToDownload) {
			logdebug("skipping excluded path: %v", fileToDownload)
			continue
		}

		if approveEveryDownload {
			if yesNo(fmt.Sprintf("Would you like to download '%v'", fileToDownload)) {
				//
				// TODO allow to skip entire directories
				//
				filteredFiles = append(filteredFiles, fileToDownload)
			}
		} else {
			filteredFiles = append(filteredFiles, fileToDownload)
		}
	}

	return filteredFiles
}

func _argparseHelper(arg string) {
	if arg == "report-files" {
		ReadOnly = true
	} else if arg == "--debug" {
		LogLevel = 2
	} else if arg == "--info" {
		LogLevel = 1
	}
}
func argparse() {
	if len(os.Args) > 1 {
		_argparseHelper(os.Args[1])
	}
	if len(os.Args) > 2 {
		_argparseHelper(os.Args[2])
	}
}

func main() {

	// TODO cleanup: use envVars struct
	envVars := os.Environ()
	home := ""
	user := ""
	remoteIP := ""
	sshUser := ""
	sshKey := ""
	for _, env_var := range envVars {
		switch {
		case strings.HasPrefix(env_var, "HOME="):
			home = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "USER="):
			user = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_REMOTE_ADDRESS="):
			remoteIP = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_USER="):
			sshUser = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_KEY="):
			sshKey = strings.Split(env_var, "=")[1]
		}
	}

	argparse()

	tmpExcludedDirs := strings.Split(read(path.Join(home, "Documents/config/video-syncer-excluded-dirs.conf")), "\n")
	// remove empty string = last element
	tmpExcludedDirs = tmpExcludedDirs[:len(tmpExcludedDirs)-1]

	var excludedDirs []string = nil
	for _, exclude := range tmpExcludedDirs {
		if exclude != "" {
			excludedDirs = append(excludedDirs, exclude)
		}
	}
	var excludedFilenames []string
	excludedFilenames = append(excludedFilenames, ".DS_Store")
	excludedFilenames = append(excludedFilenames, ".envrc")
	prettyPrintArray("DEBUG", "excludedFilenames", excludedFilenames)

	syncFileContentsLinux := read(path.Join(home, "Documents/misc/videos", "videos-home.txt"))
	syncFileContentsDarwin := read(path.Join(home, "Documents/misc/videos", "videos-work.txt"))
	// strip mpv commands
	syncFileContentsLinux = strings.Split(syncFileContentsLinux, "\n\n")[0]
	// strip mpv commands
	syncFileContentsDarwin = strings.Split(syncFileContentsDarwin, "\n\n")[0]

	filesToSyncLinux := strings.Split(syncFileContentsLinux, "\n")[1:]
	filesToSyncDarwin := strings.Split(syncFileContentsDarwin, "\n")[1:]

	var filesToSync []string = nil
	var rsyncInfoPtr *RsyncInfo
	_ = rsyncInfoPtr
	var directoryInfo *DirectoryInfo
	_ = directoryInfo

	if len(sshUser) > 0 {
		rsyncInfoPtr = &RsyncInfo{
			RemoteIP: remoteIP,
			SshUser:  sshUser,
			SshKey:   sshKey,
		}
	}
	if runtime.GOOS != "linux" {
		// if linux use the darwin sync contents
		// and vice-versa
		filesToSync = filesToSyncLinux

		// TODO duplicated
		//      do not hardcode video locations
		directoryInfo = &DirectoryInfo{
			LocalVideoDirectory:  "/Users/" + user + "/Movies",
			RemoteVideoDirectory: "/home/" + sshUser + "/Videos",
		}

	} else {
		filesToSync = filesToSyncDarwin

		// TODO duplicated
		//      do not hardcode video locations
		directoryInfo = &DirectoryInfo{
			LocalVideoDirectory:  "/home/" + user + "/Videos",
			RemoteVideoDirectory: "/Users/" + sshUser + "/Movies",
		}

	}

	// logdebug("GOOS: %#v", runtime.GOOS)
	// prettyPrintArray("DEBUG", "filesToSync", filesToSync)

	askAboutDeletions := false
	approveEveryDownload := false
	if !ReadOnly {
		askAboutDeletions = yesNo("Would you like to ask about deletions?")
		approveEveryDownload = yesNo("Would you like to approve every download?")
	}

	logdebug("askAboutDeletions:%v", askAboutDeletions)
	logdebug("approveEveryDownload:%v", approveEveryDownload)

	filesVisited, err := walkPath(directoryInfo.LocalVideoDirectory, excludedDirs, excludedFilenames, filesToSync, askAboutDeletions, yesNo)
	if err != nil {
		logerr("walkPath error: %v", err)
	}

	if ReadOnly {
		for _, fileVisited := range filesVisited {
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

	var actualFilesToDownload []string = cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames, approveEveryDownload)

	prettyPrintArray("DEBUG", "filesVisited", filesVisited)
	// prettyPrintArray("DEBUG", "filesToDownload", filesToDownload)
	prettyPrintArray("DEBUG", "actualFilesToDownload", actualFilesToDownload)

	for _, fileToDownload := range actualFilesToDownload {
		doDownload(fileToDownload, home, directoryInfo, rsyncInfoPtr)
	}
}
