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
	"path"
	"strconv"
	"strings"
)

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

func getStartTime(home, filename string) float64 {
	fileToRead := path.Join(home, filename)
	startTimeStr := strings.Split(readContent(fileToRead), "=")[1]
	startTimeStr = strings.Split(startTimeStr, "\n")[0]
	// fmt.Printf("%s\n", startTimeStr)
	startTime, _ := strconv.ParseFloat(startTimeStr, 8)
	// fmt.Printf("%s\n", error)
	return startTime
}

func main() {
	env_vars := os.Environ()
	home := ""
	for _, env_var := range env_vars {
		// fmt.Printf("env_var: %v\n", env_var)
		switch {
		case strings.HasPrefix(env_var, "HOME="):
			home = strings.Split(env_var, "=")[1]
		}
	}
	// startTime := getStartTime(home, ".config/mpv/watch_later/85C7E7264F3A4583BD74B2AB59E6C48B")
	startTime := getStartTime(home, ".local/state/mpv/watch_later/85C7E7264F3A4583BD74B2AB59E6C48B")
	fmt.Printf("%f\n", startTime)

	cmd := exec.Command(home+"/Documents/golang/tools/video-syncer/video-syncer", home+"/Videos", "report-files")
	reportedFilesBytes, _ := cmd.Output()
	reportedFiles := string(reportedFilesBytes)
	// reportedFiles = strings.ReplaceAll(reportedFiles, "\n", "")

	splitStr := strings.Split(reportedFiles, "\n")
	// prettyPrintArray("debug", "splitStr", splitStr)
	// fmt.Printf("%#v\n", splitStr)

	for _, file := range splitStr {
		// fmt.Printf("%s\n", file)
		// TODO we have to generate checksums for mac and linux -> reverse filename from checksums
		data := []byte(home + "/Videos/" + file)
		fmt.Printf("%s\n", data)
		md5sum := md5.Sum(data)
		md5sumStr := hex.EncodeToString(md5sum[:])
		md5sumStr = strings.ToUpper(md5sumStr)
		fmt.Printf("%s\n", md5sumStr)
	}
}
