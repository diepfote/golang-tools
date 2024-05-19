# Sync MPV watch-later files

Syncs `~/.config/mpv/watch_later` or `~/.local/state/mpv/watch_later` directories between 2 computers.
Times are updated based on longest time watched like so:

```text
$ ./sync-video-syncer-mpv-watch-later-files
[INFO]: Mode: default
[INFO]: would override time for `/Maybe at some Point/jonathan blow/Compiler programming livestreams/56 Scope speed-up, part 4-5iS_-mIONqc.mp4`. cur local: 234.833333 cur remote: 3986.333333
```

*Sidenote*: To perform replacement use: `--no-dry-run`


It also able to create a mapping file to show which files we actually watched (mpv only exposes md5 hashes for filepaths):

```text
$ ./sync-video-syncer-mpv-watch-later-files create-mapping-file
[INFO]: Mode: create-mapping-file

$ cat ~/.config/mpv/watch_later/mapping.txt
filename: Darktable/07 GIMP retouch： Golden hair portrait (part 2)-Zw2CUmfbeHE.mp4
time: 00:01:23

filename: Maybe at some Point/jonathan blow/Compiler programming livestreams/56 Scope speed-up, part 4-5iS_-mIONqc.mp4
time: 00:03:54

```

This uses functionality found in [video-syncer](#youtube-video-syncer) and 
[report-videos.sh](https://github.com/diepfote/scripts/blob/fc09c10453e8527e3fb53a3c379b128310c60b69/normal-privileges_systemd_scripts/report-videos.sh)
