# Snappy

## Overview

Automatically increase your Time Machine snapshot frequency.

## Structure

```text
main.go             CLI entry point
cmd/                Cobra command definitions
internal/           Internal packages
bin/snappy          Original bash TUI (reference implementation)
docs/plans/         Design plans and decision records
```

## Development

Build:

```bash
make build
```

Run tests:

```bash
make test
```

Lint and vet:

```bash
make vet
make fmt
```

See all targets:

```bash
make help
```
