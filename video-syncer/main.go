package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

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

func doDownload(fileToDownload, home string, directoryInfo *DirectoryInfo, rsyncInfoPtr *RsyncInfo) {
	log_info("downloading: %v", fileToDownload)

	downloadUrl := getDownloadUrl(fileToDownload)
	if rsyncInfoPtr == nil && len(downloadUrl) == 0 {
		log_info("[WARNING]: downloadUrl empty. Not downloading!")
		return
	}

	directoryToSyncTo := directoryInfo.LocalVideoDirectory + "/" + filepath.Dir(fileToDownload)
	fileBase := filepath.Base(fileToDownload)
	log_info("downloading to DIR: %v", directoryToSyncTo)

	// Create dir if it does not exist
	err := os.MkdirAll(directoryToSyncTo, 0755)
	if err != nil {
		log_err("Mkdir: %v", err)
	}

	cmd := exec.Command("youtube-dl", "--add-metadata", "-i", "-f", "22", downloadUrl)
	if rsyncInfoPtr != nil {
		cmd = exec.Command(home+"/Documents/scripts/video-syncer-rsync-helper.sh", rsyncInfoPtr.SshKey, rsyncInfoPtr.SshUser+"@"+rsyncInfoPtr.RemoteIP+":"+directoryInfo.RemoteVideoDirectory+"/"+fileToDownload, directoryToSyncTo+"/"+fileBase)
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
			log_err("rsync: %v", err)
		} else {
			log_err("yt-dlp: %v", err)
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
	re := regexp.MustCompile(`^[A-z0-9]{2,6}\.\]*([A-z0-9-]{11})(\[|-){1}`)
	regexSubmatches := re.FindStringSubmatch(reverse(fileToSync))

	if len(regexSubmatches) < 3 {
		// debug("regexSubmatches < 3 for %#v. returning empty string. %#v.", fileToSync, regexSubmatches)
		return ""
	}
	// debug("regexSubmatches %#v", regexSubmatches)
	youtubeId := reverse(regexSubmatches[1])
	log_info("youtubeId: `%v` (%#v)", youtubeId, fileToSync)
	downloadUrl := ""
	if len(youtubeId) > 0 {
		downloadUrl = "https://youtu.be/" + youtubeId
	}

	return downloadUrl
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

func walkPath(localVideoDirName string, excludedDirs, excludedFilenames, filesToSync []string, askAboutDeletions bool, yesNo func(string) bool) ([]string, error) {
	var filesVisited []string

	debug("walk from %v", localVideoDirName)
	prettyPrintArray("DEBUG", "excludedDirs", excludedDirs)

	err := filepath.Walk(localVideoDirName, func(_path string, fileinfo os.FileInfo, err error) error {
		if err != nil {
			log_err("prevent panic by handling failure accessing a path %q: %v", _path, err)
			return err
		}

		if fileinfo.IsDir() && stringInArray(excludedDirs, _path) {
			debug("skipping excluded path: %v", _path)
			return filepath.SkipDir
		} else if fileinfo.IsDir() {
			debug("skipping directory (but we will look into its files): %v", _path)
			return nil
		} else if stringInArray(excludedFilenames, filepath.Base(_path)) {
			debug("skipping excluded filename: %v", _path)
			return nil
		}
		// else {
		// 	debug("not skipping path: %v", _path)
		// }

		pathWithoutLocalVideoDir := strings.Split(_path, localVideoDirName+"/")[1]
		filesVisited = append(filesVisited, pathWithoutLocalVideoDir)

		if !stringInArrayCheckForIntegerPrefixes(filesToSync, _path) && !ReadOnly {
			if askAboutDeletions {
				//
				// @TODO allow to skip entire directories
				//
				answer := yesNo("Would you like to remove '" + _path + "'")

				if answer {
					log_info("removing: %v", _path)
					err := os.Remove(_path)
					if err != nil {
						log_err("%v", err)
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

func stringInArrayCheckForIntegerPrefixes(arr []string, str string) bool {
	// prettyPrintArray("DEBUG", "stringInArrayCheckForIntegerPrefixes", arr)
	// debug("stringInArraysCheckForIntegerPrefixes str: %v", str)

	for _, element := range arr {

		// debug("element: %v = str: %v?", element, str)

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
		// debug("elementWithoutPath: %#v", elementWithoutPath)
		// debug("elementWithoutPathMatches: %#v", elementWithoutPathMatches)

		elementTail := ""
		if len(elementWithoutPathMatches) > 0 {
			// remove prepended integer
			elementTail = strings.SplitN(elementWithoutPath, " ", 2)[1]
		} else {
			elementTail = elementWithoutPath
		}

		if strings.HasPrefix(tail, elementTail) {
			// debug("\t\ttail: %v", tail)
			// debug("\telementTail: %v", elementTail)
			return true
		}

	}
	return false
}

// Nicked from https://siongui.github.io/2018/03/14/go-set-difference-of-two-arrays/
func getArrayDiff(a, b []string, rsyncInfoPtr *RsyncInfo) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			if rsyncInfoPtr == nil && len(getDownloadUrl(item)) <= 0 {
				log_info("Will not ask if `%v` should be downloaded (no youtube id)", item)
				continue
			}
			diff = append(diff, item)
		}
	}
	return
}

func cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames []string, approveEveryDownload bool) (filteredFiles []string) {
	//
	// @Cleanup: for loop is ugly. should this logic not live elsewhere?
	//           or is this not already taken care of in "reporting"
	//

	for _, fileToDownload := range filesToDownload {
		if stringInArray(excludedFilenames, fileToDownload) {
			debug("filename excluded, not syncing: %v", fileToDownload)
			continue
		} else if stringInArray(filesVisited, fileToDownload) {
			debug("file seen, not syncing: %v", fileToDownload)
			continue
		}
		info, err := os.Stat(fileToDownload)
		if err != nil {
			debug("%v", err)

			// Do not skip, if the file does not exist
			// we want to sync it.
		} else {
			// Skip directories

			if info.IsDir() {
				debug("not syncing '%v'. This is a directory.", fileToDownload)
				continue
			}
			// else {
			// 	// we might want to continue a sync --> fall through
			// }
		}

		if stringInArray(excludedDirs, fileToDownload) {
			debug("skipping excluded path: %v", fileToDownload)
			continue
		}

		if approveEveryDownload {
			if yesNo(fmt.Sprintf("Would you like to download '%v'", fileToDownload)) {
				//
				// @TODO allow to skip entire directories
				//
				filteredFiles = append(filteredFiles, fileToDownload)
			}
		} else {
			filteredFiles = append(filteredFiles, fileToDownload)
		}
	}

	return filteredFiles
}

// @TODO use the `flag` pkg
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

	// @Cleanup: use envVars struct
	envVars := os.Environ()

	home := ""
	user := ""
	remoteIP := ""
	sshUser := ""
	sshKey := ""
	// @Cleanup use a command line option instead of
	//          an env var
	reverseSync := ""

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
		case strings.HasPrefix(env_var, "REVERSE_SYNC="):
			reverseSync = strings.Split(env_var, "=")[1]
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

	filesToSyncLinux := strings.Split(syncFileContentsLinux, "\n")
	filesToSyncDarwin := strings.Split(syncFileContentsDarwin, "\n")

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
	if runtime.GOOS == "linux" && len(reverseSync) > 0 {
		filesToSync = filesToSyncDarwin

		// @TODO duplicated
		//      do not hardcode video locations
		directoryInfo = &DirectoryInfo{
			LocalVideoDirectory:  "/home/" + user + "/Videos",
			RemoteVideoDirectory: "/Users/" + sshUser + "/Movies",
		}

	} else {

		// if linux use the darwin sync contents
		// and vice-versa
		filesToSync = filesToSyncLinux

		// @TODO duplicated
		//      do not hardcode video locations
		directoryInfo = &DirectoryInfo{
			LocalVideoDirectory:  "/Users/" + user + "/Movies",
			RemoteVideoDirectory: "/home/" + sshUser + "/Videos",
		}
	}

	// debug("GOOS: %#v", runtime.GOOS)
	prettyPrintArray("DEBUG", "filesToSync after fs read", filesToSync)

	askAboutDeletions := false
	approveEveryDownload := false
	if !ReadOnly {
		askAboutDeletions = yesNo("Would you like to ask about deletions?")
		approveEveryDownload = yesNo("Would you like to approve every download?")
	}

	debug("askAboutDeletions:%v", askAboutDeletions)
	debug("approveEveryDownload:%v", approveEveryDownload)

	filesVisited, err := walkPath(directoryInfo.LocalVideoDirectory, excludedDirs, excludedFilenames, filesToSync, askAboutDeletions, yesNo)
	if err != nil {
		log_err("walkPath error: %v", err)
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
		filesToDownload = getArrayDiff(filesToSyncLinux, filesToSyncDarwin, rsyncInfoPtr)
	} else {
		filesToDownload = getArrayDiff(filesToSyncDarwin, filesToSyncLinux, rsyncInfoPtr)
	}

	var actualFilesToDownload []string = cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames, approveEveryDownload)

	prettyPrintArray("DEBUG", "filesVisited", filesVisited)
	prettyPrintArray("DEBUG", "filesToDownload", filesToDownload)
	prettyPrintArray("DEBUG", "actualFilesToDownload", actualFilesToDownload)

	for _, fileToDownload := range actualFilesToDownload {
		doDownload(fileToDownload, home, directoryInfo, rsyncInfoPtr)
	}
}
