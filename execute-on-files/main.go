package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

		if strings.Contains(fileNoSpace, "$HOME") {
			// Unexpaned variable for Home
			fileNoSpace = strings.Replace(fileNoSpace, "$HOME", home, 1)
		}

		// @TODO we would need custom logic for this to work,
		//       the glob pkg does not handle it.
		//       e.g. https://github.com/gobwas/glob
		//            // create glob with pattern-alternatives list
		//            g = glob.MustCompile("{cat,bat,[fr]at}")
		//            g.Match("cat") // true
		//            g.Match("bat") // true
		//            g.Match("fat") // true
		//            g.Match("rat") // true
		//            g.Match("at") // false
		//            g.Match("zat") // false
		// isCurlyBraceExpansion := strings.Contains(repoNoSpace, "{")
		isWildCard := strings.Contains(fileNoSpace, "*")
		if isWildCard {
			matches, err := filepath.Glob(fileNoSpace)
			if err == nil {
				for _, match := range matches {
					files = append(files, match)
				}
			}
		} else {
			files = append(files, fileNoSpace)
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
		output, err := cmd.Output()

		if err != nil {
			log_err("Worker %d: `%s %s`: %v in '%s'\n%s", workerId, Command, strings.Join(args, " "), err, workingDir, output)
			continue
		}

		header := "--\nFinished:'" + file + "'\n"
		if DisableHeader {
			header = ""
		}
		if len(output) < 1 {
			fmt.Printf("%s", header)
		} else {
			fmt.Printf("%s%s\n", header, output)
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
