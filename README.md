# Snappy

Automatically increase your Time Machine snapshot frequency.

Snappy requires **macOS**. It relies on `tmutil` and APFS snapshots, which are
only available on Apple systems.

## Installation

### Homebrew

```sh
brew install cboone/tap/snappy
```

### From source

```sh
go install github.com/cboone/snappy@latest
```

### Shell script

```sh
curl -fsSL https://raw.githubusercontent.com/cboone/snappy/main/install.sh | bash
```

To install a specific version:

```sh
curl -fsSL https://raw.githubusercontent.com/cboone/snappy/main/install.sh | bash -s -- --version v1.0.0
```

### From release

Download a binary from the [releases page](https://github.com/cboone/snappy/releases).

### Build locally

```sh
git clone https://github.com/cboone/snappy.git
cd snappy
make build
./bin/snappy
```

## Usage

```sh
snappy
```

### Key Controls

| Key | Action                       |
| --- | ---------------------------- |
| `s` | Create a snapshot            |
| `r` | Refresh snapshot list        |
| `a` | Toggle auto-snapshots on/off |
| `q` | Quit                         |

### Configuration

Snappy reads configuration from `~/.config/snappy/config.yaml` or environment
variables prefixed with `SNAPPY_`. Pass `--config <path>` to use a custom file.

| Setting                  | Env var                         | Default   | Description                        |
| ------------------------ | ------------------------------- | --------- | ---------------------------------- |
| `refresh`                | `SNAPPY_REFRESH`                | `60s`     | How often to refresh snapshot list |
| `mount`                  | `SNAPPY_MOUNT`                  | `/`       | Mount point to monitor             |
| `log_dir`                | `SNAPPY_LOG_DIR`                | (auto)    | Log directory path                 |
| `log_max_size`           | `SNAPPY_LOG_MAX_SIZE`           | `5242880` | Max log file size in bytes (5 MB)  |
| `log_max_files`          | `SNAPPY_LOG_MAX_FILES`          | `3`       | Number of rotated backup files     |
| `auto_enabled`           | `SNAPPY_AUTO_ENABLED`           | `true`    | Enable auto-snapshots at startup   |
| `auto_snapshot_interval` | `SNAPPY_AUTO_SNAPSHOT_INTERVAL` | `60s`     | Interval between auto-snapshots    |
| `thin_age_threshold`     | `SNAPPY_THIN_AGE_THRESHOLD`     | `600s`    | Age before snapshots are thinned   |
| `thin_cadence`           | `SNAPPY_THIN_CADENCE`           | `300s`    | Minimum gap kept when thinning     |

## License

[MIT License](./LICENSE). TL;DR: Do whatever you want with this software, just keep the copyright notice included. The authors aren't liable if something goes wrong.
