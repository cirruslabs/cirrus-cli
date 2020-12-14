# Persistent worker mode

CLI can run in a persistent worker mode and receive tasks from the Cirrus Cloud. This allows you to go beyond the [cloud offerings](https://cirrus-ci.org/guide/supported-computing-services/) and use your own infrastructure for running cloud tasks.

## Installation

Follow the instruction in the [INSTALL.md](https://github.com/cirruslabs/cirrus-cli/blob/master/INSTALL.md).

## Running

The simplest invocation looks like this:

```
cirrus worker run --token <poll registration token>
```

This will start the persistent worker that periodically poll for new tasks in the foreground mode.

By default, the worker's name is equal to `hostname`. Specify `--name` to explicitly provide the worker's name:

```
cirrus worker run --token <poll registration token> --name z390-worker
```

Note that persistent worker's name should be unique within a pool.

## Configuration

Path to the YAML configuration can be specified via the `--file` (or `-f` for short version) command-line flag.

Example configuration:

```yaml
token: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

name: "MacMini-Slot-1"

labels:
  device: iPhone12ProMax
```

Currently, configuration files support the same set of options exposed via the command-line flags, but in the future the only way to configure certain options would be using the configuration file.

## Writing tasks

Here's an example of how to run a task on one of the persistent workers [registered in the dashboard](https://cirrus-ci.com/):

```yaml
task:
  persistent_worker:
    labels:
      os: darwin
  script: echo "running on-premise"
```
