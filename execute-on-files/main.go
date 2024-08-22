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
var Args = []string{}

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

	flag.Parse()

	LogLevel = *logLevelPtr
	ConfigPath = *configPathPtr
	Args = flag.Args()
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

	var wg sync.WaitGroup

	if len(Args) < 1 {
		flag.Usage()
		os.Exit(0)
	}
	command := Args[0]
	Args = Args[1:]

	for _, file := range files {

		args := Args
		args = append(args, file)

		wg.Add(1)
		// @TODO
		// limit the number of concurrent go routines/green threads
		// "worker pool pattern"
		// https://www.perplexity.ai/search/golang-limit-amount-of-green-t-331ynhdAQpu7fTUsK9mDkQ#1
		//
		go func(command string, args []string) {
			defer wg.Done()
			workingDir, _ := os.Getwd()
			log_info("Running: %s args:%v in '%s'", command, args, workingDir)
			cmd := exec.Command(command, args...)
			// cmd.Dir = workingDir
			output, err := cmd.Output()
			if err != nil {
				log_err("%s %v: %v in '%s'", command, args, err, workingDir)
				return
			}
			if len(output) < 1 {
				fmt.Printf("Finished:'%s'\n--\n", file)
			} else {
				fmt.Printf("Finished:'%s'\n%s--\n", file, output)
			}
		}(command, args)
	}

	wg.Wait()
}
