package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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
var Timeout time.Duration = 3 * time.Second
var ShowHeader = true
var IsRepos = true

func worker(workerId int, finished chan<- struct{}, paths <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range paths {
		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		if !IsRepos {
			ctx, cancel = context.WithCancel(context.Background())
		}
		defer cancel()

		// we use cmd.Dir instead (git -C ...)
		// path_arg := []string{"-C", path}
		// Args := append(path_arg, Args...)

		if !IsRepos {
			path_arg := []string{path}
			Args = append(path_arg, Args...)
			debug("Running: `%s %s`", Command, strings.Join(Args, " "), path)
		}
		cmd := exec.CommandContext(ctx, Command, Args...)
		if IsRepos {
			cmd.Dir = path
			debug("Running: `%s %s` in '%s'", Command, strings.Join(Args, " "), path)
		}
		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()
		cmd.Start()

		// Read stdout and stderr, concurrently
		stdoutBytesCh := make(chan []byte)
		stderrBytesCh := make(chan []byte)
		errCh := make(chan error, 2)

		asyncReadPipe(stdoutPipe, "stdout", errCh, stdoutBytesCh, ctx)
		asyncReadPipe(stderrPipe, "stderr", errCh, stderrBytesCh, ctx)

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

		finished <- struct{}{}
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log_err("Timeout %s exceeded: `%s %s`: %v in '%s'", Timeout, Command, strings.Join(Args, " "), err, path)
			} else {
				log_err("`%s %s`: %v in '%s'. stderr: %s", Command, strings.Join(Args, " "), err, path, stderrOutput)
			}
			continue
		}

		if ShowHeader {
			if len(stdoutOutput) < 1 {
				fmt.Printf("--\nFinished:'%s'\n", path)
			} else {
				fmt.Printf("--\nFinished:'%s'\n%s", path, stdoutOutput)
			}
		} else {
			if len(stdoutOutput) > 1 {
				fmt.Printf("%s", stdoutOutput)
			}
		}
	}
}

func getPaths(home, config_name string) []string {
	fpath := ""

	if filepath.IsAbs(config_name) {
		fpath = config_name
	} else {
		config_folder := ".config/personal"
		fpath = path.Join(home, config_folder, config_name)
	}
	pathsFileContent := read(fpath)

	tmp_paths := strings.Split(pathsFileContent, "\n")

	paths := []string{}
	for _, path := range tmp_paths {
		pathNoSpace := strings.TrimSpace(path)

		if len(pathNoSpace) < 1 {
			// Empty lines
			continue

		} else if strings.HasPrefix(pathNoSpace, "#") {
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
		fields, _ := shell.Fields(pathNoSpace, env)
		for _, field := range fields {
			matches, err := filepath.Glob(field)
			if err == nil {
				for _, match := range matches {
					paths = append(paths, match)
				}
			}
		}
	}
	return paths
}

func argparse() {

	// info to display: [INFO]: INFO: actualFilesToDownload%!(EXTRA string=[...
	LogLevel = 1
	Color = true
	ConfigFilename = "repo.conf"
	NumWorkers = 4

	args := os.Args[1:]
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--":
			continue
		case "--loglevel":
			v, _ := strconv.Atoi(args[i+1])
			LogLevel = v
			i++
		case "-t", "--timeout":
			v, _ := strconv.Atoi(args[i+1])
			Timeout = time.Duration(v) * time.Second
			i++
		case "-w", "--max-concurrent-tasks":
			v, _ := strconv.Atoi(args[i+1])
			NumWorkers = v
			i++
		case "--no-header":
			ShowHeader = false
		case "--no-color":
			Color = false
		case "--files":
			IsRepos = false
		case "-c", "--config":
			ConfigFilename = args[i+1]
			i++
		default:
			// treat as positional arg
			positional = append(positional, arg)
		}
	}

	if len(positional) < 1 {
		fmt.Println("usage: [options] <command> [args]")
		os.Exit(0)
	}

	debug("positional: %v", positional)
	Command = positional[0]
	Args = positional[1:]
}

func main() {
	argparse()
	// enable color in git output
	args := []string{}
	if Color {
		if Command == "git" {
			args = append(args, "-c")
			args = append(args, "color.status=always")
		} else if Command == "grep" {
			args = append(args, "--color=always")
		}
	}
	if Command == "grep" {
		args = append(args, "--exclude-dir=.git")
		args = append(args, "--exclude-dir=.helm")
		args = append(args, "--exclude-dir=.tox")
		args = append(args, "--exclude-dir=.pulumi")
		args = append(args, "--exclude-dir=.cache")
		args = append(args, "--exclude-dir=.mypy_cache")
		args = append(args, "--exclude-dir=.eggs")
		args = append(args, "--exclude-dir=*.egg-info")
		args = append(args, "--exclude-dir=*venv*")
		args = append(args, "--exclude-dir=_build")
		args = append(args, "--exclude-dir=__pycache__")
		args = append(args, "--exclude-dir=.ruff_cache")
		args = append(args, "--exclude=\"*.pyc\"")
		args = append(args, "--exclude-dir=.pytest_cache")
		args = append(args, "--exclude=poetry.lock")
		args = append(args, "--exclude-dir=htmlcov")
		args = append(args, "--exclude=\"*.html\"")
		args = append(args, "--exclude=build.*trace")
		args = append(args, "--exclude=Session.vim")
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

	paths := getPaths(home, ConfigFilename)
	prettyPrintArray("DEBUG", "paths to work on", paths)

	log_info("number of paths: %d", len(paths))

	numChannels := NumWorkers
	log_info("number of channels: %d", numChannels)
	jobs := make(chan string, NumWorkers)
	numPaths := len(paths)
	finished := make(chan struct{}, numPaths)

	var wg sync.WaitGroup
	log_info("number of workers: %d", NumWorkers)

	if IsRepos {
		log_info("timeout: %ds", Timeout/time.Second)
	}

	go func() {
		tasks_done := 0
		tasks_remaining := 0
		for range finished {
			tasks_done++

			if !ShowHeader {
				if tasks_done%10 == 0 {
					if numPaths > 0 {
						tasks_remaining = numPaths - tasks_done
					}
					log_info("remaining tasks: %d", tasks_remaining)
				}
			}
		}
	}()

	for id := 1; id <= NumWorkers; id++ {
		wg.Add(1)
		go worker(id, finished, jobs, &wg)
		debug("added worker: %d", id)
	}

	for _, path := range paths {
		jobs <- path
		debug("added path: %s", path)
	}

	close(jobs)
	wg.Wait()
	close(finished)
}
