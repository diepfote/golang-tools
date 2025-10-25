package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mvdan.cc/sh/v3/shell"
)

var Color bool
var ConfigFilename string
var Command string
var Args = []string{}
var NumWorkers int
var timeout time.Duration = 3 * time.Second

func worker(workerId int, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for repo := range jobs {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// we use cmd.Dir instead (git -C ...)
		// repo_arg := []string{"-C", repo}
		// Args := append(repo_arg, Args...)

		debug("Running: `%s %s` in '%s'", Command, strings.Join(Args, " "), repo)
		cmd := exec.CommandContext(ctx, Command, Args...)
		cmd.Dir = repo
		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()
		cmd.Start()

		// Read stdout and stderr, concurrently
		stdoutBytesCh := make(chan []byte)
		stderrBytesCh := make(chan []byte)
		errCh := make(chan error, 2)

		// Read stdout in a goroutine
		go func(ctx context.Context) {
			b, err := io.ReadAll(stdoutPipe)
			if err != nil {
				if ctx.Err() != context.DeadlineExceeded {
					errCh <- fmt.Errorf("Failed to read stdout: %w", err)
				} else {
					/* hacky way to prevent printing errors
					   if we run into a timeout
					   part 1 - stdout
					*/
					errCh <- fmt.Errorf("")
				}
			} else {
				stdoutBytesCh <- b
			}
		}(ctx)

		// Read stderr in a goroutine
		go func(ctx context.Context) {
			b, err := io.ReadAll(stderrPipe)
			if err != nil {
				if ctx.Err() != context.DeadlineExceeded {
					errCh <- fmt.Errorf("Failed to read stderr: %w", err)
				} else {
					/* hacky way to prevent printing errors
					   if we run into a timeout
					   part 1 - stderr
					*/
					errCh <- fmt.Errorf("")
				}
			} else {
				stderrBytesCh <- b
			}
		}(ctx)

		// we can no longer use cmd.Output():
		// * would not provide stderr
		// * blocks until command exits (in other words ignores timeout)
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
				/* hacky way to prevent printing errors
				   if we run into a timeout
				   part 2

				   @TODO make less "clever"
				   I tested not sending any value. Then we get a deadlock
				*/
				if len(e.Error()) > 0 {
					log_err("%v", e)
				}
				n++
			}
		}

		stdoutOutput := string(stdoutBytes)
		stderrOutput := string(stderrBytes)

		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log_err("Timeout %s exceeded: `%s %s`: %v in '%s'", timeout, Command, strings.Join(Args, " "), err, repo)
			} else {
				log_err("`%s %s`: %v in '%s'. stderr: %s", Command, strings.Join(Args, " "), err, repo, stderrOutput)
			}
			continue
		}

		if len(stdoutOutput) < 1 {
			fmt.Printf("--\nFinished:'%s'\n", repo)
		} else {
			fmt.Printf("--\nFinished:'%s'\n%s", repo, stdoutOutput)
		}
	}
}

func getRepos(home, config_name string) []string {
	fpath := ""

	if filepath.IsAbs(config_name) {
		fpath = config_name
	} else {
		config_folder := ".config/personal"
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

		env := func(name string) string {
			switch name {
			case "HOME":
				return home
			}
			return "" // leave the rest unset
		}
		fields, _ := shell.Fields(repoNoSpace, env)
		for _, field := range fields {
			isWildCard := strings.Contains(field, "*")
			if isWildCard {
				matches, err := filepath.Glob(field)
				if err == nil {
					for _, match := range matches {
						repos = append(repos, match)
					}
				}
			} else {
				repos = append(repos, field)
			}
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
	// enable color in git output
	args := []string{}
	if Command == "git" && Color {
		args = []string{"-c", "color.status=always"}
	}
	Args = append(args, Args...)

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

	log_info("number of repos: %d", len(repos))

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

	for _, repo := range repos {
		jobs <- repo
		debug("added repo: %s", repo)
	}
	close(jobs)

	wg.Wait()
}
