# Snappy

Automatically increase your Time Machine snapshot frequency.

## Installation

Clone the repository:

```bash
git clone https://github.com/cboone/snappy.git
```

## Usage

Run the interactive snapshot manager:

```bash
bin/snappy
```

The terminal UI auto-refreshes every 60 seconds. Between refreshes, use these keyboard controls:

- <kbd>s</kbd> -- create a new local snapshot
- <kbd>r</kbd> -- force-refresh the snapshot list
- <kbd>d</kbd> -- delete the oldest snapshot
- <kbd>q</kbd> -- quit

### Environment variables

| Variable        | Default                  | Purpose                              |
| --------------- | ------------------------ | ------------------------------------ |
| `SNAPPY_REFRESH` | `60`                     | Seconds between auto-refresh cycles  |
| `SNAPPY_MOUNT`  | `/`                      | Volume mount point to query          |
| `SNAPPY_LOG_DIR` | `~/.local/share/snappy`  | Log file directory                   |
| `TRACE`         | (unset)                  | Set to any value to enable bash trace |

Example with custom refresh interval:

```bash
SNAPPY_REFRESH=30 bin/snappy
```

### Log file

Snappy logs all events (snapshot creation, automatic removal, errors) to `~/.local/share/snappy/snappy.log`. This persistent log is the primary tool for observing when macOS purges snapshots under disk pressure.

## License

[MIT License](./LICENSE). TL;DR: Do whatever you want with this software, just keep the copyright notice included. The authors aren't liable if something goes wrong.
