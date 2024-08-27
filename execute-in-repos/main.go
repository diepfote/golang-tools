package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

var Color bool
var ConfigFilename string
var Command string
var Args = []string{}
var NumWorkers int

func worker(workerId int, jobs <-chan string, wg *sync.WaitGroup) {

	defer wg.Done()

	for repo := range jobs {
		// we use cmd.Dir instead (git -C ...)
		// repo_arg := []string{"-C", repo}
		// Args := append(repo_arg, Args...)

		debug("Running: %s %v in '%s'", Command, Args, repo)
		cmd := exec.Command(Command, Args...)
		cmd.Dir = repo
		output, err := cmd.Output()
		if err != nil {
			log_err("%s %v: %v in '%s'", Command, Args, err, repo)
			return
		}
		if len(output) < 1 {
			fmt.Printf("Finished:'%s'\n--\n", repo)
		} else {
			fmt.Printf("Finished:'%s'\n%s--\n", repo, output)
		}
	}
}

func getRepos(home, config_name string) []string {
	fpath := ""

	if filepath.IsAbs(config_name) {
		fpath = config_name
	} else {
		config_folder := "Documents/config"
		fpath = path.Join(home, config_folder, config_name)
	}
	reposFileContent := read(fpath)

	tmp_repos := strings.Split(reposFileContent, "\n")

	repos := []string{}
	for _, repo := range tmp_repos {
		repoNoSpace := strings.TrimSpace(repo)

		if len(repoNoSpace) < 1 {
			// Empty lines
			continue

		} else if strings.HasPrefix(repoNoSpace, "#") {
			// Comments
			continue
		}

		if strings.Contains(repoNoSpace, "$HOME") {
			// Unexpaned variable for Home
			repoNoSpace = strings.Replace(repoNoSpace, "$HOME", home, 1)
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
		isWildCard := strings.Contains(repoNoSpace, "*")
		if isWildCard {
			matches, err := filepath.Glob(repoNoSpace)
			if err == nil {
				for _, match := range matches {
					repos = append(repos, match)
				}
			}
		} else {
			repos = append(repos, repoNoSpace)
		}
	}
	return repos
}

func argparse() {
	// info to display: [INFO]: INFO: actualFilesToDownload%!(EXTRA string=[...
	logLevelPtr := flag.Int("loglevel", 1, "LogLevel: debug=2, info=1, error=0")

	noColorPtr := flag.Bool("nocolor", false, "if output should not contain color (only effects 'git')")
	configFilenamePtr := flag.String("config", "repo.conf", "e.g. repo.conf or work-repo.conf, but may also be an absolute path")

	numWorkersPtr := flag.Int("workers", 4, "number of goroutines to start")

	flag.Parse()
	LogLevel = *logLevelPtr
	Color = !*noColorPtr
	ConfigFilename = *configFilenamePtr
	NumWorkers = *numWorkersPtr

	Args = flag.Args()
	if len(Args) < 1 {
		flag.Usage()
		os.Exit(0)
	}

	Command = Args[0]
	Args = Args[1:]
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

	log_info("config file: %s", ConfigFilename)

	repos := getRepos(home, ConfigFilename)
	prettyPrintArray("DEBUG", "repos to work on", repos)

	// enable color in git output
	args := []string{}
	if Command == "git" && Color {
		args = []string{"-c", "color.status=always"}
	}
	Args = append(args, Args...)

	jobs := make(chan string, NumWorkers)
	var wg sync.WaitGroup

	for id := 1; id <= NumWorkers; id++ {
		wg.Add(1)
		go worker(id, jobs, &wg)
	}

	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	wg.Wait()
}
