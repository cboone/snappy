# Service command

Tests for `snappy service` subcommands.

## Service help

```scrut
$ "${SNAPPY_BIN}" service --help
Manage the snappy launchd agent that runs auto-snapshots in the background.

The service runs "snappy run" as a LaunchAgent, starting at login and
restarting automatically if it exits unexpectedly.

Usage:
  snappy service [flags]
  snappy service [command]

Available Commands:
  install     Install and start the snappy launchd agent
  log         Tail the snappy service log
  start       Start the snappy launchd agent
  status      Show the status of the snappy launchd agent
  stop        Stop the snappy launchd agent
  uninstall   Stop and remove the snappy launchd agent

Flags:
  -h, --help   help for service

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)

Use "snappy service [command] --help" for more information about a command.
```

## Service install help

```scrut
$ "${SNAPPY_BIN}" service install --help
Install and start the snappy launchd agent

Usage:
  snappy service install [flags]

Flags:
  -h, --help   help for install

Global Flags:
      --config string   config file (default: ~/.config/snappy/config.yaml)
```

## Service status shows not installed

When no service is installed, `service status` reports it.

```scrut
$ "${SNAPPY_BIN}" service status
Service: com.cboone.snappy
Status:  not installed

Run "snappy service install" to set up the background service.
```

## Service with no subcommand defaults to status

```scrut
$ "${SNAPPY_BIN}" service
Service: com.cboone.snappy
Status:  not installed

Run "snappy service install" to set up the background service.
```

## Service uninstall when not installed

```scrut
$ "${SNAPPY_BIN}" service uninstall
Service is not installed.
```
