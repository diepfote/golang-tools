package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/shell"
)

var ConfigPath string
var Command string
var Args []string
var NumWorkers int
var DisableHeader bool

func getFiles(home, configPath string) []string {
	filesContent := read(configPath)
	tmpFiles := strings.Split(filesContent, "\n")

	files := []string{}
	for _, file := range tmpFiles {
		fileNoSpace := strings.TrimSpace(file)

		if len(fileNoSpace) < 1 {
			// Empty lines
			continue

		} else if strings.HasPrefix(fileNoSpace, "#") {
			// Comments
			continue
		}

		env := func(name string) string {
			switch name {
			case "HOME":
				return home
			}
			return "" // leave the rest unset
		}
		fields, _ := shell.Fields(fileNoSpace, env)
		for _, field := range fields {
			files = append(files, field)
		}
	}
	return files
}

func argparse() {

	// info to display: [INFO]: INFO: actualFilesToDownload%!(EXTRA string=[...
	logLevelPtr := flag.Int("loglevel", 1, "LogLevel: debug=2, info=1, error=0")

	configPathPtr := flag.String("config", "", "files to work on in a newline delimited file (could be /dev/fd/xx)")

	numWorkersPtr := flag.Int("workers", 4, "number of goroutines to start")

	disableHeaderPtr := flag.Bool("no-header", false, "display header before command output")

	flag.Parse()
	LogLevel = *logLevelPtr
	ConfigPath = *configPathPtr
	NumWorkers = *numWorkersPtr
	DisableHeader = *disableHeaderPtr

	Args = flag.Args()
	if len(Args) < 1 {
		flag.Usage()
		os.Exit(0)
	}

	Command = Args[0]
	Args = Args[1:]
}

func worker(workerId int, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for file := range jobs {
		args := Args
		args = append(args, file)
		workingDir, _ := os.Getwd()
		debug("Worker %d: `%s %s` in '%s'", workerId, Command, strings.Join(args, " "), workingDir)
		cmd := exec.Command(Command, args...)
		// cmd.Dir = workingDir
		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()
		cmd.Start()

		// Read stdout and stderr, concurrently
		stdoutBytesCh := make(chan []byte)
		stderrBytesCh := make(chan []byte)
		errCh := make(chan error, 2)

		// Read stdout in a goroutine
		go func() {
			b, err := io.ReadAll(stdoutPipe)
			if err != nil {
				errCh <- fmt.Errorf("Failed to read stdout: %w", err)
			} else {
				stdoutBytesCh <- b
			}
		}()

		// Read stderr in a goroutine
		go func() {
			b, err := io.ReadAll(stderrPipe)
			if err != nil {
				errCh <- fmt.Errorf("Failed to read stderr: %w", err)
			} else {
				stderrBytesCh <- b
			}
		}()

		// we can no longer use cmd.Output():
		// * would not provide stderr
		//
		// And we did not use cmd.Run() as it closes pipes immediately
		// but cmd.Start() does not block, so we block here.
		err := cmd.Wait()

		// Collect outputs
		var stdoutBytes, stderrBytes []byte
		n := 0
		for n < 2 {
			select {
			case b := <-stdoutBytesCh:
				stdoutBytes = b
				n++
			case b := <-stderrBytesCh:
				stderrBytes = b
				n++
			case e := <-errCh:
				log_err("%v", e)
				n++
			}
		}

		stdoutOutput := string(stdoutBytes)
		stderrOutput := string(stderrBytes)

		if err != nil {
			log_err("Worker %d: `%s %s`: %v in '%s'\n%s", workerId, Command, strings.Join(args, " "), err, workingDir, stderrOutput)
			continue
		}

		header := "--\nFinished:'" + file + "'\n"
		if DisableHeader {
			header = ""
		}
		if len(stdoutOutput) < 1 {
			fmt.Printf("%s", header)
		} else {
			fmt.Printf("%s%s\n", header, stdoutOutput)
		}
	}
}

func main() {
	argparse()

	envVars := os.Environ()
	home := ""
	for _, env_var := range envVars {
		switch {
		case strings.HasPrefix(env_var, "HOME="):
			home = strings.Split(env_var, "=")[1]
		}
	}

	log_info("config file: %s", ConfigPath)

	files := getFiles(home, ConfigPath)
	prettyPrintArray("DEBUG", "files to work on", files)

	log_info("number of files: %d", len(files))

	numChannels := NumWorkers
	log_info("number of channels: %d", numChannels)
	jobs := make(chan string, NumWorkers)

	var wg sync.WaitGroup
	log_info("number of workers: %d", NumWorkers)

	for id := 1; id <= NumWorkers; id++ {
		wg.Add(1)
		go worker(id, jobs, &wg)
		debug("added worker: %d", id)
	}

	for _, file := range files {
		jobs <- file
		debug("added file: %s", file)
	}
	close(jobs)
	wg.Wait()
}
