package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var DryRun bool = true
var CreateMappingFile = false

func stripVideoFolder(file, videosFolder string) string {
	// debug("file: %s, videosFolder: %s\n", file, videosFolder)
	return strings.Split(file, videosFolder)[1]
}

func copySyncFile(localVideosFolder, localFile, localMpvWatchLaterFile, remoteVideosFolder, remoteFile, remoteMpvWatchLaterFile string, localStartTime, remoteStartTime float64) bool {

	// startTime := getStartTime(localHome, ".config/mpv/watch_later/7B76A69ECB27D3D80B4B96241C1E31EA")
	// startTime := getStartTime(localHome, ".local/state/mpv/watch_later/85C7E7264F3A4583BD74B2AB59E6C48B")
	// debug("%f\n", startTime)

	strippedLocalFile := stripVideoFolder(localFile, localVideosFolder)
	strippedRemoteFile := stripVideoFolder(remoteFile, remoteVideosFolder)
	if strippedLocalFile != strippedRemoteFile {
		return false
	}
	// debug("localFile : %s\n", strippedLocalFile)
	// debug("remoteFile: %s\n\n", strippedRemoteFile)

	debug("localStartTime : %s\n", localStartTime)
	debug("remoteStartTime: %s\n\n", remoteStartTime)

	if int(localStartTime) < int(remoteStartTime) {

		if DryRun {
			log_info("would override time for `%s`. cur local: %f cur remote: %f\n", strippedLocalFile, localStartTime, remoteStartTime)
			return true
		}

		log_info("override time for `%s`. cur local: %f cur remote: %f\n", strippedLocalFile, localStartTime, remoteStartTime)

		cmd := exec.Command("cp", remoteMpvWatchLaterFile, localMpvWatchLaterFile)
		output, error := cmd.Output()
		if error == nil {
			return true
		}
		log_err("[!] cp error: %s\n", output)
		return false
	}
	debug("`%s`. cur local: %f cur remote: %f\n", strippedLocalFile, localStartTime, remoteStartTime)
	return false
}

func createMD5toFilenameMappingFile(md5MappingPath, md5filepath string, fpathOffset int, fpath string, startTime float64) {

	// ignore watch_later config files if startTime does not diverge from start
	if startTime == 0.0 {
		return
	}

	content := "filename: " + fpath[fpathOffset:] + "\n"

	duration := time.Duration(startTime) * time.Second // 1 hour in seconds
	formattedDuration := fmt.Sprintf("%02d:%02d:%02d", int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
	content += "time: " + formattedDuration + "\n"

	youtubeUrl := getYoutubeUrl(fpath)
	content += "youtube: " + youtubeUrl + "\n\n"

	var file *os.File
	if _, err := os.Stat(md5MappingPath); errors.Is(err, os.ErrNotExist) {
		// file does not exist
		file, err = os.Create(md5MappingPath)
		if err != nil {
			log_err("Failed to create file %v: %v", fpath, err)
			return
		}
	} else {
		file, err = os.OpenFile(md5MappingPath, os.O_APPEND|os.O_WRONLY, 0644)
	}
	defer file.Close()

	_, err := file.WriteString(content)
	if err != nil {
		log_err("Write failed: %v", err)
		return
	}

}

func getMd5Hash(data []byte) string {
	// debug("%s\n", data)
	md5sum := md5.Sum(data)
	md5sumStr := hex.EncodeToString(md5sum[:])
	md5sumStr = strings.ToUpper(md5sumStr)
	return md5sumStr
}

func getStartTime(filename string) float64 {
	// debug("fileToRead: %s\n", filename)
	content := read(filename)

	startTime := 0.0
	if len(content) > 0 {
		startTimeStr := strings.Split(content, "=")[1]
		startTimeStr = strings.Split(startTimeStr, "\n")[0]
		// debug("%s\n", startTimeStr)
		startTime, _ = strconv.ParseFloat(startTimeStr, 8)
		// log_err("%s\n", error)
	}
	return startTime
}

func _argparseHelper(arg string) {
	if arg == "--no-dry-run" {
		DryRun = false
	} else if arg == "--debug" {
		LogLevel = 2
	} else if arg == "--info" {
		LogLevel = 1
	} else if arg == "--error" {
		LogLevel = 0
	} else if arg == "create-mapping-file" {
		CreateMappingFile = true
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
	if len(os.Args) > 3 {
		_argparseHelper(os.Args[1])
	}
}

func main() {
	argparse()

	macVideosFolder := "Movies"
	macHome := "/Users/florian.sorko"
	linuxHome := "/home/flo"
	linuxVideosFolder := "Videos"
	linuxMpvWatchLaterDir := linuxHome + "/.local/state/mpv/watch_later"
	macMpvWatchLaterDir := macHome + "/.config/mpv/watch_later"

	macSyncMpvDir := macHome + "/.config/personal/sync-config/videos"
	linuxSyncMpvWatchLaterDir := macSyncMpvDir + "/arch-mpv-watch_later"

	linuxSyncMpvDir := linuxHome + "/.config/personal/sync-config/videos"
	macSyncMpvWatchLaterDir := linuxSyncMpvDir + "/mac-mpv-watch_later"

	localHome := linuxHome
	localVideosFolder := linuxVideosFolder
	localMpvWatchLaterDir := linuxMpvWatchLaterDir
	remoteHome := macHome
	remoteVideosFolder := macVideosFolder
	remoteSyncMpvWatchLaterDir := macSyncMpvWatchLaterDir
	localSyncMpvDir := linuxSyncMpvDir
	if runtime.GOOS != "linux" {
		localHome = macHome
		localVideosFolder = macVideosFolder
		localMpvWatchLaterDir = macMpvWatchLaterDir
		remoteHome = linuxHome
		remoteVideosFolder = linuxVideosFolder
		remoteSyncMpvWatchLaterDir = linuxSyncMpvWatchLaterDir
		localSyncMpvDir = macSyncMpvDir
	}

	cmd := exec.Command(localHome+"/Repos/golang/tools/video-syncer/video-syncer", "report-files")
	reportedFilesBytes, _ := cmd.Output()
	reportedFiles := string(reportedFilesBytes)
	// reportedFiles = strings.ReplaceAll(reportedFiles, "\n", "")

	splitStr := strings.Split(reportedFiles, "\n")
	splitStr = splitStr[:len(splitStr)-1]
	// prettyPrintArray("DEBUG", "splitStr", splitStr)
	// debug("%#v\n", splitStr)

	var md5MappingPath string = ""
	if CreateMappingFile {
		log_info("Mode: create-mapping-file")
		filename := "mapping.txt"
		md5MappingPath = localSyncMpvDir + "/" + filename
		_ = os.Remove(md5MappingPath)
	} else {
		log_info("Mode: default")
	}

	for _, file := range splitStr {
		// debug("%s\n", file)
		localData := []byte(localHome + "/" + localVideosFolder + "/" + file)
		remoteData := []byte(remoteHome + "/" + remoteVideosFolder + "/" + file)
		localMd5sumStr := getMd5Hash(localData)
		remoteMd5sumStr := getMd5Hash(remoteData)

		localFile := string(localData)
		remoteFile := string(remoteData)

		localMpvWatchLaterFile := localMpvWatchLaterDir + "/" + localMd5sumStr
		remoteMpvWatchLaterFile := remoteSyncMpvWatchLaterDir + "/" + remoteMd5sumStr
		localStartTime := getStartTime(localMpvWatchLaterFile)
		remoteStartTime := getStartTime(remoteMpvWatchLaterFile)

		if CreateMappingFile {
			createMD5toFilenameMappingFile(md5MappingPath, localMpvWatchLaterFile, len(localHome+"/"+localVideosFolder+"/"), localFile, localStartTime)
		} else {
			copySyncFile(localVideosFolder, localFile, localMpvWatchLaterFile, remoteVideosFolder, remoteFile, remoteMpvWatchLaterFile, localStartTime, remoteStartTime)
		}

		// if _, err := os.Stat(string(localData)); err == nil {
		// 	debug("local: %s\n", string(localData))
		// 	debug("local: %s\n", localMd5sumStr)
		// 	debug("remote: %s\n", string(remoteData))
		// 	debug("remote: %s\n", remoteMd5sumStr)
		// }
		// else if os.IsNotExist(err) {
		// 	// path/to/whatever does *not* exist

		// }
	}
}
