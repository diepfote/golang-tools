# Execute in repos

Run arbitrary commands in git repos/plain old directories simultaneously/concurrently.
Or run them on files.


*Hint*: There is no order to how we output stderr/stdout, whichever go routine finishes first goes first.

## Options and Args

```text
Error: usage: execute [options] [flags] -- <args>
  options:
    -w/--max-concurrent-tasks <num> [default: 4]
    -c/--config <file/fd> [default: repo.conf]
    -t/--timeout <seconds> [default: 3]
  flags:
    --no-color ... disable color for `git` and `grep`  [default: colored]
    --no-header ... will report remaining tasks to stderr every 10 tasks
```

## Examples

```text
$ ~/Documents/golang/tools/execute/execute --config <(find ~/.tmux/plugins/  -maxdepth 1 -mindepth 1 -type d) -- ls
/home/flo/.tmux/plugins/tmux-continuum
CHANGELOG.md
continuum.tmux
CONTRIBUTING.md
docs
LICENSE.md
README.md
scripts

/home/flo/.tmux/plugins/tmux-resurrect
CHANGELOG.md
CONTRIBUTING.md
docs
lib
LICENSE.md
README.md
resurrect.tmux
run_tests
save_command_strategies
scripts
strategies
tests
video

```

or

```text
$ ~/Documents/golang/tools/execute-in-repos/execute-in-repos git status -sb
[INFO]: config file: repo.conf
/home/flo/Documents/dockerfiles
## master...origin/master

/home/flo/.vim
## master...origin/master

/home/flo/Documents/golang/tools
## master...origin/master
M  .gitignore
M  Makefile
A  execute/Makefile
```
