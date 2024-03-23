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

Note that by default a persistent worker has the privileges of the user that invoked it. Read more [about isolation](#isolation) below to learn how to limit or extend persistent worker privileges.

## Configuration

Path to the YAML configuration can be specified via the `--file` (or `-f` for short version) command-line flag.

Example configuration:

```yaml
token: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

name: "MacMini-Rack-1-Slot-2"

labels:
  connected-device: iPhone12ProMax

resources:
  iphone-devices: 1
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
* `rotate-size` — rotate the log file if it reaches the specified size, e.g. 640 KB or 100 MiB (defaults to no rotation)
* `max-rotations` — how many already rotated files to keep (defaults to no limit)

Example:

```yaml
log:
  level: warning
  file: cirrus-worker.log
  rotate-size: 100 MB
  max-rotations: 10
```

### Resource management

Persistent Worker supports resource management, [similarly to Kubernetes](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/), but in a slightly more simplified way.

Resources are key-value pairs, where key represents an arbitrary resource name, and value is a floating-point number specifying how many of that resource the worker has.

When scheduling tasks, Cirrus CI ensures that all the tasks the worker receives do not exceed the resources defined in it's configuration file, for example, with the following configuration:

```yaml
token: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

name: "mac-mini-1"

resources:
  tart-vms: 2
```

...a worker won't run more than 2 tasks simultaneously from the following `.cirrus.yml`:

```yaml
persistent_worker:
  isolation:
    tart:
      image: ghcr.io/cirruslabs/macos-monterey-base:latest
      user: admin
      password: admin

  resources:
    tart-vms: 1

task:
  name: Test
  script: make test

task:
  name: Build
  script: make build

task:
  name: Release
  script: make release
```

### Security

#### Restricting possible isolation environments

By default, Persistent Worker allows running tasks with [any isolations](#isolation), which is roughly equivalent to the following configuration:

```yaml
security:
  allowed-isolations:
    none: {}
    container: {}
    tart: {}
    vetu: {}
```

To only allow running tasks inside of [Tart VMs](https://github.com/cirruslabs/tart), for example, specify the following in your Persistent Worker configuration:

```yaml
security:
  allowed-isolations:
    tart: {}
```

#### Restricting Tart images

Further, you can also restrict which Tart VM images can be used (wildcard character `*` is supported), and force Softnet to enable [better network isolation](https://github.com/cirruslabs/softnet#working-model):

```yaml
security:
  allowed-isolations:
    tart:
      allowed-images:
        - "ghcr.io/cirruslabs/*"
      force-softnet: true
```

#### Restricting Tart volumes

To restrict which volume paths can be mounted and in which mode, use `allowed-volumes`:

```yaml
security:
  allowed-isolations:
    tart:
      allowed-volumes:
        # Allow mounting /Volumes/SSD and all directories inside of it
        - source: "/Volumes/SSD/*"
        # Allow mounting /var/src in read-only mode, but not directories inside of it
        - source: "/var/src"
          force-readonly: true
```

#### Restricting Vetu images

Similarly to Tart, you can also restrict which Vetu VM images can be used (wildcard character `*` is supported):

```yaml
security:
  allowed-isolations:
    vetu:
      allowed-images:
        - "ghcr.io/cirruslabs/*"
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

## Isolation

### No isolation

By default, a persistent worker does not isolate execution of a task. All the task instructions are executed directly on
the worker which can have side effects. This is intended since the main use case for persistent workers is to test on
bare metal.

Note that the user that starts the Persistent Worker is the user under which the task will run. You may create a separate user with limited access to limit tasks privileges, or conversely grant tasks access to the whole machine by running the Persistent Worker as `root`:

```
sudo cirrus worker run --token <poll registration token>
```

### Container

To use this isolation type, install and configure a container engine like [Docker](https://github.com/cirruslabs/cirrus-cli/blob/master/INSTALL.md#docker) or [Podman](https://github.com/cirruslabs/cirrus-cli/blob/master/INSTALL.md#podman) (essentially the ones supported by the [Cirrus CLI](https://github.com/cirruslabs/cirrus-cli)).

Here's an example that runs a task in a separate container with a couple directories from the host machine being accessible:

```yaml
persistent_worker:
  isolation:
    container:
      image: debian:latest
      cpu: 24
      memory: 128G
      volumes:
        - /path/on/host:/path/in/container
        - /tmp/persistent-cache:/tmp/cache:ro

task:
  script: uname -a
```

### Tart

To use this isolation type, install the [Tart](https://github.com/cirruslabs/tart) on the persistent worker's host machine.

Here's an example of a configuration that will run the task inside of a fresh macOS virtual machine created from a remote [`ghcr.io/cirruslabs/macos-ventura-base:latest`](https://github.com/cirruslabs/macos-image-templates/pkgs/container/macos-ventura-base) VM image:

```yaml
persistent_worker:
  isolation:
    tart:
      image: ghcr.io/cirruslabs/macos-ventura-base:latest
      user: admin
      password: admin

task:
  script: uname -a
```

Once the VM spins up, persistent worker will connect to the VM's IP-address over SSH using `user` and `password` credentials and run the latest agent version.

### Vetu

To use this isolation type, install the [Vetu](https://github.com/cirruslabs/vetu) on the persistent worker's host machine.

```yaml
persistent_worker:
  isolation:
    vetu:
      image: ghcr.io/cirruslabs/ubuntu:latest
      user: admin
      password: admin

task:
  script: uname -a
```

Once the VM spins up, persistent worker will connect to the VM's IP-address over SSH using `user` and `password` credentials and run the latest agent version.

## Standby VM

You can define a VM that the Persistent Worker will run even if no tasks are scheduled.

When an actual task is assigned to a Persistent Worker, and if the isolation specification of that task matches the defined standby configuration, the standby VM will be used to run the task instead of creating a new VM from scratch. Otherwise, the standby VM will be terminated to simplify the resource management.

This has an effect of speeding up the start times significantly, since the cloning, configuring and booting of a new VM would be normally be already done at the time the request to run a new VM arrives to a Persistent Worker.

Here's an example on how to configure a standby VM for Tart:

```yaml
standby:
  resources:
    tart-vms: 1
  isolation:
    tart:
      image: ghcr.io/cirruslabs/macos-sonoma-base:latest
      user: admin
      password: admin
      cpu: 4
      memory: 12
```

This corresponds to the following CI configuration:

```yaml
persistent_worker:
  isolation:
    tart:
      image: ghcr.io/cirruslabs/macos-sonoma-base:latest
      user: admin
      password: admin
      cpu: 4
      memory: 12288 # 13 GB
```

Currently only Tart isolation is supported.
