package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func stripVideoFolder(file, videosFolder string) string {
	// fmt.Printf("file: %s, videosFolder: %s\n", file, videosFolder)
	return strings.Split(file, videosFolder)[1]
}

func copySyncFile(localVideosFolder, localFile, localMd5sumStr, localMpvWatchLaterDir, remoteVideosFolder, remoteFile, remoteMd5sumStr, remoteSyncMpvWatchLaterDir string) bool {

	// startTime := getStartTime(localHome, ".config/mpv/watch_later/7B76A69ECB27D3D80B4B96241C1E31EA")
	// startTime := getStartTime(localHome, ".local/state/mpv/watch_later/85C7E7264F3A4583BD74B2AB59E6C48B")
	// fmt.Printf("%f\n", startTime)

	strippedLocalFile := stripVideoFolder(localFile, localVideosFolder)
	strippedRemoteFile := stripVideoFolder(remoteFile, remoteVideosFolder)
	if strippedLocalFile != strippedRemoteFile {
		return false
	}
	// fmt.Printf("[.] localFile : %s\n", strippedLocalFile)
	// fmt.Printf("[.] remoteFile: %s\n\n", strippedRemoteFile)

	localMpvWatchLaterFile := localMpvWatchLaterDir + "/" + localMd5sumStr
	remoteMpvWatchLaterFile := remoteSyncMpvWatchLaterDir + "/" + remoteMd5sumStr
	localStartTime := getStartTime(localMpvWatchLaterFile)
	remoteStartTime := getStartTime(remoteMpvWatchLaterFile)
	// fmt.Printf("[.] localStartTime : %s\n", localStartTime)
	// fmt.Printf("[.] remoteStartTime: %s\n\n", remoteStartTime)

	if int(localStartTime) < int(remoteStartTime) {
		fmt.Printf("[!] override time for `%s`. cur local: %f cur remote: %f\n", strippedLocalFile, localStartTime, remoteStartTime)

		cmd := exec.Command("cp", remoteMpvWatchLaterFile, localMpvWatchLaterFile)
		output, error := cmd.Output()
		if error == nil {
			return true
		}
		fmt.Printf("[!] mv error: %s\n", output)
		return false
	}
	fmt.Printf("[.] INFO: `%s`. cur local: %f cur remote: %f\n", strippedLocalFile, localStartTime, remoteStartTime)
	return false
}

func getMd5Hash(data []byte) string {
	// fmt.Printf("%s\n", data)
	md5sum := md5.Sum(data)
	md5sumStr := hex.EncodeToString(md5sum[:])
	md5sumStr = strings.ToUpper(md5sumStr)
	return md5sumStr
}

func prettyPrintArray(typeOfMessage, nameOfArray string, arr []string) {
	// snatched from https://stackoverflow.com/a/56242100
	s, _ := json.MarshalIndent(arr, "", "\t")
	fmt.Printf("[%s]: %s: %s\n", typeOfMessage, nameOfArray, string(s))
}

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

func getStartTime(filename string) float64 {
	// fmt.Printf("fileToRead: %s\n", filename)
	content := readContent(filename)

	startTime := 0.0
	if len(content) > 0 {
		startTimeStr := strings.Split(content, "=")[1]
		startTimeStr = strings.Split(startTimeStr, "\n")[0]
		// fmt.Printf("%s\n", startTimeStr)
		startTime, _ = strconv.ParseFloat(startTimeStr, 8)
		// fmt.Printf("%s\n", error)
	}
	return startTime
}

func main() {
	// env_vars := os.Environ()
	// home := ""
	// for _, env_var := range env_vars {
	// 	// fmt.Printf("env_var: %v\n", env_var)
	// 	switch {
	// 	case strings.HasPrefix(env_var, "HOME="):
	// 		home = strings.Split(env_var, "=")[1]
	// 	}
	// }

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

	cmd := exec.Command(localHome+"/Documents/golang/tools/video-syncer/video-syncer", localHome+"/"+localVideosFolder, "report-files")
	reportedFilesBytes, _ := cmd.Output()
	reportedFiles := string(reportedFilesBytes)
	// reportedFiles = strings.ReplaceAll(reportedFiles, "\n", "")

	splitStr := strings.Split(reportedFiles, "\n")
	splitStr = splitStr[:len(splitStr)-1]
	// prettyPrintArray("debug", "splitStr", splitStr)
	// fmt.Printf("%#v\n", splitStr)

	for _, file := range splitStr {
		// fmt.Printf("%s\n", file)
		// TODO we have to generate checksums for mac and linux -> reverse filename from checksums
		localData := []byte(localHome + "/" + localVideosFolder + "/" + file)
		remoteData := []byte(remoteHome + "/" + remoteVideosFolder + "/" + file)
		localMd5sumStr := getMd5Hash(localData)
		remoteMd5sumStr := getMd5Hash(remoteData)

		localFile := string(localData)
		remoteFile := string(remoteData)
		copySyncFile(localVideosFolder, localFile, localMd5sumStr, localMpvWatchLaterDir, remoteVideosFolder, remoteFile, remoteMd5sumStr, remoteSyncMpvWatchLaterDir)
		// if _, err := os.Stat(string(localData)); err == nil {
		// 	fmt.Printf("local: %s\n", string(localData))
		// 	fmt.Printf("local: %s\n", localMd5sumStr)
		// 	fmt.Printf("remote: %s\n", string(remoteData))
		// 	fmt.Printf("remote: %s\n", remoteMd5sumStr)
		// }
		// else if os.IsNotExist(err) {
		// 	// path/to/whatever does *not* exist

		// }
	}
}
