# Video syncer

Sync youtube videos between computers based on text files containing  
files names (relatives paths to `~/Movies` (Darwin) or `~/Videos` (Linux)).  
**Hint**: It also ensures that the directory strucuture on both machines stays the same.

Example text files:

* `~/.config/personal/sync-config/videos/videos-home.txt`:

  ```
  Some/Obli Haxx (Mid of Jan, 2014).mp4
  ebpf/Container Performance Analysis-bK9A5ODIgac.mp4
  ebpf/LISA21 - BPF Internals-_5Z2AU7QTH4.mp4
  ```

* `~/.config/personal/sync-config/videos/videos-work.txt`:

  ```
  Timothy Roscoe/Unix50 - Unix Today and Tomorrow - The Kernel-CyJ1ZCwtiRg.mp4
  ```

Example output of executable:

```
$ ~/Documents/golang/tools/video-syncer/video-syncer
[INFO]: url is: https://youtu.be/CyJ1ZCwtiRg
Would you like to remove 'Timothy Roscoe/Unix50 - Unix Today and Tomorrow - The Kernel-CyJ1ZCwtiRg.mp4' [y|N]?


[INFO]: filesToDownload: []string{"Some/Obli Haxx (Mid of Jan, 2014).mp4", "ebpf/Container Performance Analysis-bK9A5ODIgac.mp4", "ebpf/LISA21 - BPF Internals-_5Z2AU7QTH4.mp4"}
Would you like to approve every download? [y|N]?


[INFO]: syncying: Some/Obli Haxx (Mid of Jan, 2014).mp4
[WARNING]: downloadUrl empty. Not syncing!
[INFO]: syncying: ebpf/Container Performance Analysis-bK9A5ODIgac.mp4
[INFO]: url is: https://youtu.be/bK9A5ODIgac
[INFO]: syncing to DIR: ebpf
[youtube] bK9A5ODIgac: Downloading webpage
[download] Resuming download at byte 15630518
[download] Destination: Container Performance Analysis-bK9A5ODIgac.mp4
[download] 100% of 87.21MiB in 20:0352KiB/s ETA 00:00658
[youtube] bK9A5ODIgac: Downloading webpage
[download] Resuming download at byte 15630518
[download] Destination: Container Performance Analysis-bK9A5ODIgac.mp4
[download] 100% of 87.21MiB in 20:0352KiB/s ETA 00:00658


[INFO]: syncying: ebpf/LISA21 - BPF Internals-_5Z2AU7QTH4.mp4
[INFO]: url is: https://youtu.be/_5Z2AU7QTH4
[INFO]: syncing to DIR: ebpf
[youtube] _5Z2AU7QTH4: Downloading webpage
[download] Destination: LISA21 - BPF Internals-_5Z2AU7QTH4.mp4
[download] 100% of 97.70MiB in 38:1756KiB/s ETA 00:001nown ETA
[youtube] _5Z2AU7QTH4: Downloading webpage
[download] Destination: LISA21 - BPF Internals-_5Z2AU7QTH4.mp4
[download] 100% of 97.70MiB in 38:1756KiB/s ETA 00:001nown ETA
```

Resulting files:

```
$ cd ~/Movies/
~/Movies
$ ls -alh ebpf/
total 200M
drwxr-xr-x   4 florian staff 128 Jan  7 06:30  .
drwx------+ 14 florian staff 448 Jan  7 04:59  ..
-rw-r--r--   1 florian staff 88M Oct 17  2018 'Container Performance Analysis-bK9A5ODIgac.mp4'
-rw-r--r--   1 florian staff 98M Sep  4 01:18 'LISA21 - BPF Internals-_5Z2AU7QTH4.mp4'
~/Movies
$ ls -alh Timothy\ Roscoe/
total 297M
drwxr-xr-x   3 florian staff   96 Dec 31 03:53  .
drwx------+ 14 florian staff  448 Jan  7 04:59  ..
-rw-r--r--   1 florian staff 297M Oct 31  2019 'Unix50 - Unix Today and Tomorrow - The Kernel-CyJ1ZCwtiRg.mp4'
```
