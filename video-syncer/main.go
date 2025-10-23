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

	"github.com/mattn/go-shellwords"
)

var ReportOnly bool = false
var DryRun bool = true

type RsyncInfo struct {
	RemoteLocation string
	SshUser        string
	SshKey         string
}

type DirectoryInfo struct {
	LocalVideoDirectory  string
	RemoteVideoDirectory string
}

func doDownload(fileToDownload, home string, directoryInfo *DirectoryInfo, rsyncInfoPtr *RsyncInfo) {
	log_info("downloading: %v", fileToDownload)

	downloadUrl := getYoutubeUrl(fileToDownload)
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
	cmd.Dir = directoryToSyncTo
	if rsyncInfoPtr != nil {
		debug("RemoteLocation: %s", rsyncInfoPtr.RemoteLocation)
		// we use the "rsync backend" and will not hit the interwebs

		if len(rsyncInfoPtr.SshKey) > 0 &&
			len(rsyncInfoPtr.SshUser) > 0 &&
			len(rsyncInfoPtr.RemoteLocation) > 0 {
			// we need to establish a ssh connection
			//
			cmd = exec.Command(home+"/Repos/scripts/video-syncer-rsync-helper.sh", rsyncInfoPtr.SshKey, rsyncInfoPtr.SshUser+"@"+rsyncInfoPtr.RemoteLocation+":"+directoryInfo.RemoteVideoDirectory+"/"+fileToDownload, directoryToSyncTo+"/"+fileBase)
		} else {
			// we fetch from a local storage medium
			//
			cmd = exec.Command(home+"/Repos/scripts/video-syncer-rsync-helper.sh", rsyncInfoPtr.RemoteLocation+"/"+fileToDownload, directoryToSyncTo+"/"+fileBase)
		}
	} else {
		// default case:
		// we download from youtube directly
	}

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

func walkPath(localVideoDir string, excludedDirs, excludedFilenames, filesToSync []string, askAboutDeletions bool, yesNo func(string) bool) ([]string, error) {
	var filesVisited, reportFilesToDelete []string

	debug("walk from %v", localVideoDir)
	prettyPrintArray("DEBUG", "excludedDirs", excludedDirs)

	err := filepath.Walk(localVideoDir, func(_path string, fileinfo os.FileInfo, err error) error {
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

		tail := strings.Split(_path, localVideoDir+"/")[1]
		filesVisited = append(filesVisited, tail)

		if DryRun || askAboutDeletions {
			if !stringInArrayCheckForIntegerPrefixes(filesToSync, tail) {
				if DryRun {
					reportFilesToDelete = append(reportFilesToDelete, tail)
				} else {
					//3850845
					// TODO allow to skip entire directories
					//
					answer := yesNo("Would you like to remove '" + tail + "'")

					if answer {
						log_info("removing: %v", tail)
						err := os.Remove(_path)
						if err != nil {
							log_err("%v", err)
						}
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

	if DryRun {
		prettyPrintArray("INFO", "reportFilesToDelete", reportFilesToDelete)
		fmt.Println()
	}

	return filesVisited, err
}

func stringInArrayCheckForIntegerPrefixes(arr []string, str string) bool {
	// prettyPrintArray("DEBUG", "stringInArrayCheckForIntegerPrefixes", arr)
	// debug("stringInArraysCheckForIntegerPrefixes str: %v", str)

	for _, element := range arr {

		// debug("stringInArrayCheckForIntegerPrefixes: element: %v = str: %v?", element, str)

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
			if rsyncInfoPtr == nil && len(getYoutubeUrl(item)) <= 0 {
				log_info("Will not ask if `%v` should be downloaded (no youtube id)", item)
				continue
			}
			diff = append(diff, item)
		}
	}
	return
}

func removeDuplicates(s []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func getFilesOpenedByMpv(bashCmds []string) []string {
	var files []string = nil
	var words []string = nil
	var err error = nil
	for _, commandLineInput := range bashCmds {
		debug("commandLineInput: %#v", commandLineInput)
		if commandLineInput == "" {
			continue
		}

		parser := shellwords.NewParser()
		words, err = parser.Parse(commandLineInput)
		if err != nil {
			debug("shellwords parse error: %#v")
			continue
		}
		debug("words: %#v", words)
		if len(words) < 2 {
			continue
		}

		command := words[0]
		if command == "mpv" {
			file := words[1]
			if !strings.HasPrefix(file, "https://") {
				// we already keep track of files
				// in ~/Videos / ~/Movies
				continue
			}
			files = append(files, words[1])
		} else if command == "mpv-rsync.net" {
			files = append(files, "sftp://mpv-rsync.net/"+words[1])
		}
	}

	return removeDuplicates(files)
}

func cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames []string, approveEveryDownload bool) (filteredFiles []string) {
	//
	// TODO cleanup: for loop is ugly. should this logic not live elsewhere?
	//               or is this not already taken care of in "reporting"
	//

	for _, fileToDownload := range filesToDownload {
		if stringInArray(excludedFilenames, fileToDownload) {
			// debug("filename excluded, not syncing: %v", fileToDownload)
			continue

		} else if stringInArray(excludedDirs, filepath.Dir(fileToDownload)) {
			debug("This:  %v:%#v", fileToDownload, excludedDirs)
			continue
		} else if stringInArray(filesVisited, fileToDownload) {
			// debug("file seen, not syncing: %v", fileToDownload)
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

func filterFilesOnServer(files []string) []string {
	var filtered []string = nil
	for _, f := range files {
		if strings.HasPrefix(f, "https://") || strings.HasPrefix(f, "sftp://") {
			continue
		}
		filtered = append(filtered, f)
	}

	return filtered
}

// TODO use the `flag` pkg
func _argparseHelper(arg string) {
	if arg == "report-files" {
		ReportOnly = true
	} else if arg == "--no-dry-run" {
		DryRun = false
	} else if arg == "--debug" {
		LogLevel = 2
	} else if arg == "--info" {
		LogLevel = 1
	} else if arg == "--error" {
		LogLevel = 0
	}
}
func argparse() {
	// info to display: [INFO]: INFO: actualFilesToDownload%!(EXTRA string=[...
	LogLevel = 1
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
	remoteLocation := ""
	sshUser := ""
	sshKey := ""
	for _, env_var := range envVars {
		switch {
		case strings.HasPrefix(env_var, "HOME="):
			home = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "USER="):
			user = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_REMOTE_ADDRESS="):
			remoteLocation = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_USER="):
			sshUser = strings.Split(env_var, "=")[1]
		case strings.HasPrefix(env_var, "VIDEO_SYNCER_SSH_KEY="):
			sshKey = strings.Split(env_var, "=")[1]
		}
	}

	argparse()

	tmpExcludedDirs := strings.Split(read(path.Join(home, ".config/personal/video-syncer-excluded-dirs.conf")), "\n")
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
	excludedFilenames = append(excludedFilenames, "Makefile")
	excludedFilenames = append(excludedFilenames, "list.txt")
	excludedFilenames = append(excludedFilenames, "missing.txt")
	prettyPrintArray("DEBUG", "excludedFilenames", excludedFilenames)

	syncFileContentsLinux := read(path.Join(home, ".config/personal/sync-config/videos", "videos-home.txt"))

	syncFileContentsDarwin := read(path.Join(home, ".config/personal/sync-config/videos", "videos-work.txt"))

	filesToSyncLinux := filterFilesOnServer(strings.Split(syncFileContentsLinux, "\n"))
	filesToSyncDarwin := filterFilesOnServer(strings.Split(syncFileContentsDarwin, "\n"))

	bashCmds := strings.Split(read(path.Join(home, ".bash_history_x")), "\n")
	mpvFilesOpened := getFilesOpenedByMpv(bashCmds)

	var filesToSync []string = nil
	var rsyncInfoPtr *RsyncInfo
	_ = rsyncInfoPtr
	var directoryInfo *DirectoryInfo
	_ = directoryInfo

	if len(remoteLocation) > 0 {
		rsyncInfoPtr = &RsyncInfo{
			RemoteLocation: remoteLocation,
			SshUser:        sshUser,
			SshKey:         sshKey,
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

	if len(filesToSync) > 1 {
		filesToSyncLastElement := filesToSync[len(filesToSync)-1]
		if len(filesToSyncLastElement) < 1 {
			// remove last element if  empty
			filesToSync = filesToSync[:len(filesToSync)-1]
		}
	}

	// debug("GOOS: %#v", runtime.GOOS)
	prettyPrintArray("DEBUG", "filesToSync after fs read", filesToSync)

	askAboutDeletions := false
	approveEveryDownload := false
	if !ReportOnly && !DryRun {
		askAboutDeletions = yesNo("Would you like to ask about deletions?")
		approveEveryDownload = yesNo("Would you like to approve every download?")
	}

	debug("askAboutDeletions:%v", askAboutDeletions)
	debug("approveEveryDownload:%v", approveEveryDownload)

	filesVisited, err := walkPath(directoryInfo.LocalVideoDirectory, excludedDirs, excludedFilenames, filesToSync, askAboutDeletions, yesNo)
	if err != nil {
		log_err("walkPath error: %v", err)
	}

	if ReportOnly {
		for _, fileVisited := range filesVisited {
			fmt.Printf("%v\n", fileVisited)
		}
		for _, f := range mpvFilesOpened {
			fmt.Printf("%v\n", f)
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

	debug("filesToDownload: %#v", filesToDownload)
	var actualFilesToDownload []string = cleanupFilesToDownload(filesToDownload, filesVisited, excludedDirs, excludedFilenames, approveEveryDownload)

	fmt.Println()
	prettyPrintArray("DEBUG", "filesVisited", filesVisited)
	prettyPrintArray("DEBUG", "filesToDownload", filesToDownload)
	if DryRun {
		prettyPrintArray("INFO", "actualFilesToDownload", actualFilesToDownload)
		return
	} else {
		prettyPrintArray("DEBUG", "actualFilesToDownload", actualFilesToDownload)
	}

	for _, fileToDownload := range actualFilesToDownload {
		doDownload(fileToDownload, home, directoryInfo, rsyncInfoPtr)
	}
}
