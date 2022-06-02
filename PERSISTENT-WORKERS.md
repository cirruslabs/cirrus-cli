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

### Parallels Desktop for Mac

If your host has [Parallels Desktop](https://www.parallels.com/products/desktop/) installed, then a persistent worker
can execute tasks in available Parallels VMs (worker will clone a VM, run the task and then remove the temporary cloned
VM).

Here is an example of how to instruct a persistent worker to use Parallels isolation:

```yaml
task:
  persistent_worker:
    labels:
      os: darwin
    isolation:
      paralllels:
        image: big-sur-xcode # locally registered VM
        # username and password for SSHing into the VM to start a task
        user: admin
        password: admin
        platform: darwin # VM platform. Only darwin is supported at the moment.
  script: echo "running on-premise in a Parallels VM"
```

**Note**: Persistent worker supports packed Parallels VMs (it will unpack such VMs first before cloning). This can help
with the process of updating VMs on persistent workers. If one need to update a VM named `VM_NAME` which is already unpacked
on the host at `~/Parallels/VM_NAME.pvm`, then you can simply distribute a packed `VM_NAME.pvmp` file to `~/Parallels/VM_NAME.pvmp`
and run a script similar to the one below to update the VM almost atomically:

```bash
prlctl unregister VM_NAME
rm -rf ~/Parallels/VM_NAME.pvm
prlctl register ~/Parallels/VM_NAME.pvmp
```
