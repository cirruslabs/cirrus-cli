# Persistent Worker Mode

CLI can run in a persistent worker mode and receive tasks from the Cirrus Cloud. This allows you to go beyond the [cloud offerings](https://cirrus-ci.org/)
and use your own infrastructure for running cloud tasks. The main use case is to run Cirrus tasks directly on a hardware
without any isolation and not in a virtual ephemeral environment.

## Installation

Follow the instruction in the [INSTALL.md](INSTALL.md) but note that Docker or Podman installation is not required.

## Running

The simplest invocation looks like this:

```
cirrus worker run --token <poll registration token>
```

This will start the persistent worker that periodically poll for new tasks in the foreground mode.

By default, the worker's name is equal to the name of the current system. Specify `--name` to explicitly provide the worker's name:

```
cirrus worker run --token <poll registration token> --name z390-worker
```

Note that persistent worker's name should be unique within a pool.

## Configuration

Path to the YAML configuration can be specified via the `--file` (or `-f` for short version) command-line flag.

Example configuration:

```yaml
token: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

name: "MacMini-Rack-1-Slot-2"

labels:
  connected-device: iPhone12ProMax
```

Currently configuration files support the same set of options exposed via the command-line flags, while the rest of the options can only be configured via configuration file and are documented here.

### Reserved Labels

Worker automatically populates the following lables:

* `os` — `GOOS` of the CLI binary (for example, `linux`, `windows`, `darwin`, `solaris`, etc.).
* `arch` — `GOARCH` of the CLI binary (for example, `amd64`, `arm64`, etc.).
* `version` — CLI's version.
* `hostname` — host name reported by the host kernel.
* `name` — worker name passed via `--name` flag of the YAML configuration. Defaults to `hostname` with stripped common suffixes like `.local` and `.lan`.

### Logging

Can be configured in the `log` section of the configuration file. The following things can be customized:

* `level` — logging level to use, either `panic`, `fatal`, `error`, `warning`, `info`, `debug` or `trace` (defaults to `info`)
* `file` — log to the specified file instead of terminal
* `rotate-every` — rotate the log file if it reaches the specified size, e.g. 640 KB or 100 MiB (defaults to no rotation)
* `max-rotations` — how many already rotated files to keep (defaults to no limit)

Example:

```yaml
log:
  level: warning
  file: cirrus-worker.log
  rotate-every: 100 MB
  max-rotations: 10
```

## Writing tasks

Here's an example of how to run a task on one of the persistent workers [registered in the dashboard](https://cirrus-ci.com/):

```yaml
task:
  persistent_worker:
    labels:
      os: darwin
      arch: arm64
  script: echo "running on-premise"
```
