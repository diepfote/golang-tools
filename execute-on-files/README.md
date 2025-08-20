# Execute on files

Run arbitrary commands on files simultaneously/concurrently.

`-config`  
Files are specified by `-config`. It contains filenames separate by newlines, wildcards are also allowed.  
`-config` ... can be command redirect (a filedescriptor) `<(find -type d -name '*something*')` or a regular file.

*Hint*: There is no order to how we output command output for files, whichever go routine finishes first wins.

## Speed comparison


amount of files:

```text
/r/m/f/1TB/scanlime-in-progress-new-channel
$ ls | wc -l
205
```

### find

```text
$ time find -name '*.mp4' -exec bash -c 'stat "$0"; ffprobe-get-date "$0"' {} \;
...
...
20180414
  File: ./._Does it even MBR？？？ Let's try DOS or Linux on the AVC Edge again, now with Sigrok + bus probe-ako_wEfj9gs.mp4
  Size: 4096            Blocks: 256        IO Block: 4096   regular file
Device: 8,2     Inode: 17816       Links: 1
Access: (0777/-rwxrwxrwx)  Uid: ( 1000/     flo)   Gid: ( 1000/     flo)
Access: 2024-08-23 07:47:52.000000000 +0000
Modify: 2024-08-23 07:47:52.000000000 +0000
Change: 2024-08-23 07:47:52.000000000 +0000
 Birth: -

real    2m27,942s
user    0m45,595s
sys     1m39,264s
```

### execute-on-files

contents of `/tmp/it`:

```text
$ cat /tmp/it
#!/usr/bin/env bash
ffprobe-get-date "$1"
stat "$1"
```

--

```text
/r/m/f/1TB/scanlime-in-progress-new-channel
$ time ~/Documents/golang/tools/execute-on-files/execute-on-files -config <(ls *.mp4) /tmp/it
...
...
'Unbox Pinebook and e-paper ⧸ Designing a simple FPGA GPU ⧸ Can we 720p with iCEBreaker？ [2018-11-23]-whrFOIMcK8w.mp4':pwd:'/run/media/flo/1TB/scanlime-in-progress-new-channel'
20181123
  File: Unbox Pinebook and e-paper ⧸ Designing a simple FPGA GPU ⧸ Can we 720p with iCEBreaker？ [2018-11-23]-whrFOIMcK8w.mp4
  Size: 2134490042      Blocks: 4168960    IO Block: 4096   regular file
Device: 8,2     Inode: 14237       Links: 1
Access: (0777/-rwxrwxrwx)  Uid: ( 1000/     flo)   Gid: ( 1000/     flo)
Access: 2024-08-23 12:19:27.000000000 +0000
Modify: 2018-11-27 21:50:57.000000000 +0000
Change: 2018-11-27 21:50:57.000000000 +0000
 Birth: -


real    0m39,174s
user    1m9,154s
sys     2m46,024s
```

## Examples

### Basic

```text
./execute-on-files -config <(ls *.mp4) /tmp/it
./execute-on-files -config <(ls *.mp4) ffprobe-get-date
./execute-on-files -config <(ls *.mp4) stat
./execute-on-files -config <(find -name '*network reor*.mp4') stat
./execute-on-files -config <(ls *.m4a) ffmpeg-dynamic-range-compress-file

$ ./execute-on-files -config <(echo main.go) ls -alh
[INFO]: config file: /dev/fd/63
'main.go':pwd:'/home/flo/Documents/golang/tools/execute-on-files'
-rw-r--r-- 1 flo flo 3,0K Aug 23 13:34 main.go

$ ./execute-on-files -config <(echo main.go) stat
[INFO]: config file: /dev/fd/63
'main.go':pwd:'/home/flo/Documents/golang/tools/execute-on-files'
  File: main.go
  Size: 3019            Blocks: 8          IO Block: 4096   regular file
Device: 0,47    Inode: 1911786     Links: 1
Access: (0644/-rw-r--r--)  Uid: ( 1000/     flo)   Gid: ( 1000/     flo)
Access: 2024-08-23 12:02:30.754224342 +0000
Modify: 2024-08-23 13:34:23.153871516 +0000
Change: 2024-08-23 13:34:23.157204802 +0000
 Birth: 2024-08-23 12:02:30.754224342 +0000
```

### More sophisticated

command to run across directories:

```text
$ execute-on-files -config <(find "$PWD" -type d) /tmp/command.sh
```

#### Sum the duration of all `mp4` files in a directory

`/tmp/command.sh`:

```text
# plain old. slow
for f in *; do ffprobe-get-duration "$f" | awk -F '.' '{ printf("%s seconds + ", $1); }'; done | qalc

# OR

# fast
$ cat /tmp/it
#!/usr/bin/env bash
ffprobe-get-duration "$1" | awk -F '.' '{ printf("%s seconds + ", $1); }' >> /tmp/all
/r/m/f/1TB/scanlime-in-progress-new-channel

$ execute-on-files -config <(find -name '*.mp4') /tmp/it && qalc < /tmp/all
```

#### Merge audio files into one (across several directories)

`/tmp/command.sh`:

```text
#!/usr/bin/env bash

# all of these stem from https://www.shellcheck.net/wiki/
set -o pipefail  # propagate errors
set -u  # exit on undefined
set -e  # exit on non-zero return value
#set -f  # disable globbing/filename expansion
shopt -s failglob  # error on unexpaned globs
shopt -s inherit_errexit  # Bash disables set -e in command substitution by default; reverse this behavior

set -x
dir="$1"
shift

temp="$(mktemp -d)"
cleanup () { rm -rf "$temp"; }
trap cleanup EXIT

config="$temp"/config.txt


for f in "$dir"/*.m4a; do
  echo "file '$f'" >> "$config"
done

name="$(basename "$dir")"
out="$temp/How It All Ends: $name.m4a"
pwd
echo "$temp"
cat "$config"
ffmpeg -f concat -safe 0 -i "$config" -c copy "$out"
mv "$out" "$dir"
```
