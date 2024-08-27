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
var Args = []string{}

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

	flag.Parse()

	LogLevel = *logLevelPtr
	Color = !*noColorPtr
	ConfigFilename = *configFilenamePtr
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

	log_info("config file: %s", ConfigFilename)

	repos := getRepos(home, ConfigFilename)
	prettyPrintArray("DEBUG", "repos to work on", repos)

	var wg sync.WaitGroup

	if len(Args) < 1 {
		flag.Usage()
		os.Exit(0)
	}
	command := Args[0]
	Args = Args[1:]

	args := []string{}
	if command == "git" && Color {
		args = []string{"-c", "color.status=always"}
	}
	args = append(args, Args...)
	for _, repo := range repos {

		// we use cmd.Dir instead
		// repo_arg := []string{"-C", repo}
		// all_args := append(repo_arg, args...)
		all_args := args

		wg.Add(1)
		go func(command string, args []string) {
			defer wg.Done()
			debug("Running: %s %v in '%s'", command, args, repo)
			cmd := exec.Command(command, args...)
			cmd.Dir = repo
			output, err := cmd.Output()
			if err != nil {
				log_err("%s %v: %v in '%s'", command, args, err, repo)
				return
			}
			if len(output) < 1 {
				fmt.Printf("Finished:'%s'\n--\n", repo)
			} else {
				fmt.Printf("Finished:'%s'\n%s--\n", repo, output)
			}
		}(command, all_args)
	}

	wg.Wait()
}
