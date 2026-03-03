# Snappy

![Snappy the Swamp Protector vs The Rusty Clanker](./docs/images/scary-snappy-4x1.jpg)

[![Go Report Card](https://img.shields.io/badge/go%20report%20card-A+-important?style=for-the-badge&labelColor=f3eecd&color=81834a)](https://goreportcard.com/report/github.com/cboone/snappy) [![GitHub branch check runs](https://img.shields.io/github/check-runs/cboone/snappy/main?style=for-the-badge&label=tests&labelColor=f3eecd&color=81834a)](https://github.com/cboone/snappy/actions) ![macOS 11+](https://img.shields.io/badge/macOS-11+-critical?style=for-the-badge&labelColor=f3eecd&color=b18655) [![MIT License](https://img.shields.io/github/license/cboone/snappy?style=for-the-badge&labelColor=f3eecd&color=b18655)](./LICENSE)

[**Quick Start**](#quick-start) | [**Why Use Snappy?**](#why-use-snappy) | [**Usage**](#commands-and-options) | [**Restoring Files and Snapshots**](#restoring-files-and-snapshots) | [**Limitations**](#limitations) | [**Other Tools**](#comparison)

**Frequent, automatic, super fast, lightweight snapshot backups of your entire drive.** Snappy uses the macOS built-in snapshotting system to allow easy access and rollbacks to individual files, directories, or the entire disk.

`brew install cboone/tap/snappy` and it's automatically installed and running. Run `snappy` on its own to see the status of your snapshots, view your config, and mount your snapshot backups to restore files.

Snappy only runs on macOS. It relies on [`tmutil`](https://ss64.com/mac/tmutil.html) and [APFS snapshots](https://eclecticlight.co/2026/01/31/explainer-snapshots-2/). If you're on Linux, [`zsh-auto-snapshot`](https://manpages.debian.org/trixie/zfs-auto-snapshot/zfs-auto-snapshot.8.en.html) is a good option (and is what inspired Snappy in the first place). AFAICT, it should work on pretty much any macOS version since 11 (macOS Big Sur), when [Time Machine](<https://en.wikipedia.org/wiki/Time_Machine_(macOS)>) began to use APFS snapshotting.

**You don't need to use Time Machine for Snappy to work.** You can use Time Machine's fancy UI to view your backups and restore files if you want, or you can use Snappy's snapshot mounting to browse and restore your files via the Finder. None of the functionality requires Time Machine to be enabled, even using the Time Machine UI to view backups.

You barely even need Snappy, for that matter. It provides easy setup, handy config options (with good defaults), a TUI to view and manage your snapshots, commands to mount your snapshots, and a few other niceties. But at its core, Snappy's a glorified Bash script cron job. So much so that I've included a super simple Bash script you could use instead, if you want frequent snapshots without installing a binary: [`snappy-ez`](./bin/snappy-ez). More details on [how to use it](#snappy-ez) below.

**AI usage:** Snappy is a 3 out of 5 on [my personal vibes scale](#TODO), meaning that the code and tests were written by LLMs micro-managed by me. Robot code; human architecture, design (in all senses), code review, and manual testing. I wrote this README; other docs are a mix. Read more about my use of LLMs and my workflow in [my AI transparency statement](#TODO).

## Why Use Snappy?

To add to your local hard drive's safety net. By default, Snappy tells macOS to take a complete snapshot of your drive every minute, then thins those snapshots down to 5 minute increments at 10 minutes back. macOS thins backups older than 1 hour to hourly, then reduces the frequency to daily after 1 day, and weekly after 1 week.

```text
now            -10 min          -1 hour          -1 day           -1 week
 ├────────────────┼────────────────┼────────────────┼────────────────┼────╶╶╶╶╶
 │||||||||||||||||│ |  |  |  |  |  |   |   |    |   │        |       │
 │  every minute  │  every 5 min   │     hourly     │      daily     │  weekly
 ╰──────────── Snappy ─────────────┴───────────── macOS ─────────────┴────╶╶╶╶╶
```

These snapshots are complete clones of your hard drive, stored exactly as it was at the moment the snapshot was taken. Because of the copy-on-write cloning technique they use, minimal storage is needed for each one. Megabytes or gigabytes per clone on a many-terabyte file system, typically. This is what allows you to save so many copies of everything on your drive.

Hopefully you'll install Snappy and let it run and rarely think about it. But if something goes wrong, like an LLM agent deleting what it shouldn't, a ransomware attack encrypting your sensitive data, an installation gone bad, or just a simple mistake, you can easily restore your files or even your entire drive.

See below for more on [How Snappy Works](#how-snappy-works), [How to Configure Snapshot Frequency](#configuration), and [How to Restore Files and Snapshots](#restoring-files-and-snapshots).

## Quick Start

Install Snappy via [Homebrew](https://brew.sh):

```sh
brew install cboone/tap/snappy
```

That installs the `snappy` command, along with shell completions and a man page, and sets up Snappy to run every minute. Until something goes wrong, that's really all you need to do.

To open Snappy's TUI, just run `snappy`. You'll see what snapshots have been taken and information about them, logs of all Snappy's and macOS's snapshot management activities. You can delete or thin snapshots to clear up space. Most importantly, you can mount snapshots as read-only local drives to browse and restore files.

All of this can be done non-interactively via various [commands and options](#commands-and-options). Read more below, or run `snappy help` or `man snappy`. Also, see below for more on [How Snappy Works](#how-snappy-works), [How to Configure Snapshot Frequency](#configuration), and [How to Restore Files and Snapshots](#restoring-files-and-snapshots).

## Restoring Files and Snapshots

TODO: Write.

## Commands and Options

TODO: Write.

## Installation

TODO: Update.

### Shell script

```sh
curl -fsSL https://raw.githubusercontent.com/cboone/snappy/main/install.sh | bash
```

To install a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/cboone/snappy/main/install.sh | bash -s -- --version v1.0.0
```

### From release

```sh
git clone https://github.com/cboone/snappy.git
cd snappy
make build
./bin/snappy
```

## snappy-ez

TODO: Update.

A standalone bash script that provides snappy's core functionality (create
snapshots, thin old ones, log state) without the TUI, Go, or a build step.
Download it, edit the constants, and run.

### Download

```sh
curl -fsSL https://raw.githubusercontent.com/cboone/snappy/main/bin/snappy-ez -o snappy-ez
chmod +x snappy-ez
```

### Run in the foreground

```sh
./snappy-ez
```

Press `Ctrl-C` to stop.

### Run in the background

```sh
./snappy-ez >> ~/snappy-ez.log 2>&1 &
tail -f ~/snappy-ez.log    # monitor
kill %1                     # stop
```

### Customize

Edit the constants at the top of the script:

| Constant             | Default | Description                                   |
| -------------------- | ------- | --------------------------------------------- |
| `SNAPSHOT_INTERVAL`  | `60`    | Seconds between snapshots                     |
| `THIN_AGE_THRESHOLD` | `600`   | Snapshots younger than this are never thinned |
| `THIN_CADENCE`       | `300`   | Minimum gap between kept old snapshots        |

## Configuration

TODO: Update.

Snappy reads configuration from `~/.config/snappy/config.yaml` or environment
variables prefixed with `SNAPPY_`. Pass `--config <path>` to use a custom file.

### Generate a config file

Create a default config file with all settings and comments:

```sh
snappy config init
```

This writes to `~/.config/snappy/config.yaml` (or the path given by
`--config`). The command will not overwrite an existing file.

#### View effective configuration

Show the active configuration, including values from the config file,
environment variables, and defaults:

```sh
snappy config
```

#### Settings

| Setting                  | Env var                         | Default   | Description                        |
| ------------------------ | ------------------------------- | --------- | ---------------------------------- |
| `auto_enabled`           | `SNAPPY_AUTO_ENABLED`           | `true`    | Enable auto-snapshots at startup   |
| `auto_snapshot_interval` | `SNAPPY_AUTO_SNAPSHOT_INTERVAL` | `60s`     | Interval between auto-snapshots    |
| `auto_snapshot_interval` | `SNAPPY_AUTO_SNAPSHOT_INTERVAL` | `60s`     | Interval between auto-snapshots    |
| `log_dir`                | `SNAPPY_LOG_DIR`                | (auto)    | Log directory path                 |
| `log_dir`                | `SNAPPY_LOG_DIR`                | (auto)    | Log directory path                 |
| `log_max_files`          | `SNAPPY_LOG_MAX_FILES`          | `3`       | Number of rotated backup files     |
| `log_max_size`           | `SNAPPY_LOG_MAX_SIZE`           | `5242880` | Max log file size in bytes (5 MB)  |
| `mount`                  | `SNAPPY_MOUNT`                  | `/`       | Mount point to monitor             |
| `refresh`                | `SNAPPY_REFRESH`                | `60s`     | How often to refresh snapshot list |
| `refresh`                | `SNAPPY_REFRESH`                | `60s`     | How often to refresh snapshot list |
| `thin_age_threshold`     | `SNAPPY_THIN_AGE_THRESHOLD`     | `600s`    | Age before snapshots are thinned   |
| `thin_age_threshold`     | `SNAPPY_THIN_AGE_THRESHOLD`     | `600s`    | Age before snapshots are thinned   |
| `thin_cadence`           | `SNAPPY_THIN_CADENCE`           | `300s`    | Minimum gap kept when thinning     |
| `thin_cadence`           | `SNAPPY_THIN_CADENCE`           | `300s`    | Minimum gap kept when thinning     |

## How Snappy Works

TODO: Write.

### Time Machine's Default Behavior

```text
now            -1 hour          -1 day           -1 week
 ├────────────────┼────────────────┼────────────────┼────╶╶╶╶╶╶
 │                |   |   |    |   │        |       │
 │                │     hourly     │      daily     │  weekly
 ╰────────────────┴───────────── macOS ─────────────┴────╶╶╶╶╶╶
```

## Background

TODO: Write.

I began building it to replicate the functionality of [`zsh-auto-snapshot`](https://manpages.debian.org/trixie/zfs-auto-snapshot/zfs-auto-snapshot.8.en.html), which allows Linux users (who are using [the zfs filesystem](https://en.wikipedia.org/wiki/ZFS)) to

## Limitations

TODO: Write.

## Comparison

TODO: Write.

There are other good tools that provide similar functionality.

## Open Questions

Details I haven't yet resolved with my own experimentation and haven't found definitive answers for on the web:

- [ ] How does TM manage local snapshots when the remote backup disk isn't connected?
  - [ ] Does it keep taking hourly snapshots?
  - [ ] How does it manage thinning?
- [ ] Is a new snapshot taken every time you manually trigger a TM backup?
- [ ] Can non-system drives be handled in the same way?
- [ ] Can snapshots be transferred, à la zfs?
- [ ] Do the TM backup exclusions apply to local snapshots?
- [ ] It appears that without TM local backups enabled, macOS prevents snapshots from being kept longer than 24 hours. Is this still true with TM local backups only enabled? With remote backups only enabled?
- [ ] How long does TM keep the weekly snapshots?

## License

[MIT License](./LICENSE). TL;DR: Do whatever you want with this software, just keep the copyright notice included. The authors aren't liable if something goes wrong.
