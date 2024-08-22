# Execute in repos

Run arbitrary commands in git repos or plain old directories simultaneously/concurrently.


`-config`  
Repos are specified via `-config` [default="$HOME/Documents/repo.conf"]. It contains repo paths separate by newlines, wildcards are also allowed.  
`-config` ... can be a command redirect (a filedescriptor) `<(find -type d -name '*something*')` or a regular file.

Any git command uses color by default, use `-nocolor` to disable color for `git` commands.

*Hint*: There is no order to how we output command output for repos, whichever go routine finishes first wins.

## Examples

```text
$ ~/Documents/golang/tools/execute-in-repos/execute-in-repos -config <(find ~/.tmux/plugins/  -maxdepth 1 -mindepth 1 -type d) -- ls
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
A  execute-in-repos/Makefile
```
