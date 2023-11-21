package main

import (
	"crypto/md5"
	"encoding/hex"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func stripVideoFolder(file, videosFolder string) string {
	// debug("file: %s, videosFolder: %s\n", file, videosFolder)
	return strings.Split(file, videosFolder)[1]
}

func copySyncFile(localVideosFolder, localFile, localMd5sumStr, localMpvWatchLaterDir, remoteVideosFolder, remoteFile, remoteMd5sumStr, remoteSyncMpvWatchLaterDir string) bool {

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

	localMpvWatchLaterFile := localMpvWatchLaterDir + "/" + localMd5sumStr
	remoteMpvWatchLaterFile := remoteSyncMpvWatchLaterDir + "/" + remoteMd5sumStr
	localStartTime := getStartTime(localMpvWatchLaterFile)
	remoteStartTime := getStartTime(remoteMpvWatchLaterFile)
	debug("localStartTime : %s\n", localStartTime)
	debug("remoteStartTime: %s\n\n", remoteStartTime)

	if int(localStartTime) < int(remoteStartTime) {
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

func main() {
	// info
	LogLevel = 1

	macVideosFolder := "Movies"
	macHome := "/Users/florian.sorko"
	linuxHome := "/home/flo"
	linuxVideosFolder := "Videos"
	linuxMpvWatchLaterDir := linuxHome + "/.local/state/mpv/watch_later"
	linuxSyncMpvWatchLaterDir := macHome + "/Documents/misc/videos/arch-mpv-watch_later"
	macMpvWatchLaterDir := macHome + "/.config/mpv/watch_later"
	macSyncMpvWatchLaterDir := linuxHome + "/Documents/misc/videos/mac-mpv-watch_later"

	localHome := linuxHome
	localVideosFolder := linuxVideosFolder
	localMpvWatchLaterDir := linuxMpvWatchLaterDir
	remoteHome := macHome
	remoteVideosFolder := macVideosFolder
	remoteSyncMpvWatchLaterDir := macSyncMpvWatchLaterDir
	if runtime.GOOS != "linux" {
		localHome = macHome
		localVideosFolder = macVideosFolder
		localMpvWatchLaterDir = macMpvWatchLaterDir
		remoteHome = linuxHome
		remoteVideosFolder = linuxVideosFolder
		remoteSyncMpvWatchLaterDir = linuxSyncMpvWatchLaterDir
	}

	cmd := exec.Command(localHome+"/Documents/golang/tools/video-syncer/video-syncer", "report-files")
	reportedFilesBytes, _ := cmd.Output()
	reportedFiles := string(reportedFilesBytes)
	// reportedFiles = strings.ReplaceAll(reportedFiles, "\n", "")

	splitStr := strings.Split(reportedFiles, "\n")
	splitStr = splitStr[:len(splitStr)-1]
	// prettyPrintArray("DEBUG", "splitStr", splitStr)
	// debug("%#v\n", splitStr)

	for _, file := range splitStr {
		// debug("%s\n", file)
		localData := []byte(localHome + "/" + localVideosFolder + "/" + file)
		remoteData := []byte(remoteHome + "/" + remoteVideosFolder + "/" + file)
		localMd5sumStr := getMd5Hash(localData)
		remoteMd5sumStr := getMd5Hash(remoteData)

		localFile := string(localData)
		remoteFile := string(remoteData)
		copySyncFile(localVideosFolder, localFile, localMd5sumStr, localMpvWatchLaterDir, remoteVideosFolder, remoteFile, remoteMd5sumStr, remoteSyncMpvWatchLaterDir)
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
