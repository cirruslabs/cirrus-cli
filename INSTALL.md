# Index

* [Homebrew](#homebrew)
* [Debian-based distributions](#debian-based-distributions) (Debian, Ubuntu, etc.)
* [RPM-based distributions](#rpm-based-distributions) (Fedora, CentOS, etc.)
* [Prebuilt Binary](#prebuilt-binary)
* [Golang](#golang)
* CI integrations
  * [GitHub Actions](#github-actions)
  * [Travis CI](#travis-ci)
  * [Circle CI](#circle-ci)
  * [TeamCity](#teamcity)
  * [Cloud Build](#cloud-build)
  * [Cirrus CI](#cirrus-ci)

# Prerequisites

Since currently CLI runs all of it's tasks either via [Docker](#docker) or [Podman](#podman), make sure one of these is installed.

## Docker

[Docker](https://docker.com/) is available on a variety operating systems and distributions.

OS-specific instructions can be found here: https://docs.docker.com/get-docker/

## Podman

[Podman](https://podman.io/) is generally only available on Linux.

Distribution-specific instructions can be found here: https://podman.io/getting-started/installation

When Podman binary is found on the system it'll be used automatically, however, if there's also a Docker installed, then it will be preferred instead.

To force the CLI to use Podman backend, pass the `--container-backend=podman` flag when running a build:

```
cirrus run --container-backend=podman Lint
```

### Rootless builds

Follow the Podman's official [rootless tutorial](https://github.com/containers/podman/blob/master/docs/tutorials/rootless_tutorial.md) to configure a rootless environment.

Once this is done, you can use the Podman backend as you'd normally do, without becoming a `root`

# Installation

## Homebrew

```bash
brew install cirruslabs/cli/cirrus
```

## Debian-based distributions

Firstly, make sure that the APT transport for downloading packages via HTTPS and common X.509 certificates are installed:

```shell
sudo apt-get update && sudo apt-get -y install apt-transport-https ca-certificates
```

Then, add the Cirrus Labs repository:

```shell
echo "deb [trusted=yes] https://apt.fury.io/cirruslabs/ /" | sudo tee /etc/apt/sources.list.d/cirruslabs.list
```

Finally, update the package index files and install the Cirrus CLI:

```shell
sudo apt-get update && sudo apt-get -y install cirrus-cli
```

## RPM-based distributions

First, create a `/etc/yum.repos.d/cirruslabs.repo` file with the following contents:

```
[cirruslabs]
name=Cirrus Labs Repo
baseurl=https://yum.fury.io/cirruslabs/
enabled=1
gpgcheck=0
```

Then, install the Cirrus CLI:

```shell
sudo yum -y install cirrus-cli
```

## Prebuilt Binary

Check the [releases page](https://github.com/cirruslabs/cirrus-cli/releases) for a pre-built `cirrus` binary for your platform.

Here is a one liner for Linux/macOS to download the latest release and add

```bash
curl -L -o cirrus https://github.com/cirruslabs/cirrus-cli/releases/latest/download/cirrus-$(uname | tr '[:upper:]' '[:lower:]')-amd64 \
  && sudo mv cirrus /usr/local/bin/cirrus && sudo chmod +x /usr/local/bin/cirrus
```

## Golang

If you have latest [Golang](https://golang.org/) installed locally, you can run:

```
go install github.com/cirruslabs/cirrus-cli/...@latest
```

This will build and place the `cirrus` binary in `$GOPATH/bin`.

To be able to run `cirrus` command from anywhere, make sure the `$GOPATH/bin` directory is added to your `PATH`
environment variable (see [article in the Go wiki](https://github.com/golang/go/wiki/SettingGOPATH) for more details).

## GitHub Actions

Here is an example `.github/workflows/cirrus.yml` configuration file that runs Cirrus Tasks using CLI:

```yaml
name: Run Cirrus Tasks

on:
  push:
    branches: [ master ]
  pull_request:
    branches: [ master ]

jobs:
  cirrus:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - uses: cirruslabs/cirrus-action@v2
```

**Note:** Cirrus Action integrates natively with GitHub Actions caching mechanism by using [HTTP Caching Proxy Action](https://github.com/cirruslabs/http-cache-action)

## Travis CI

Here is an example of `.travis.yml` configuration file that runs Cirrus Tasks using CLI:

```yaml
services:
  - docker

cache:
  directories:
    - /home/travis/.cache/cirrus/

before_install:
  - curl -L -o cirrus https://github.com/cirruslabs/cirrus-cli/releases/latest/download/cirrus-linux-amd64
  - sudo mv cirrus /usr/local/bin/cirrus
  - sudo chmod +x /usr/local/bin/cirrus

script: cirrus run
```

## Circle CI

Here is an example of `.circleci/config.yml` configuration file that runs Cirrus Tasks using CLI:

```yaml
version: 2.1
jobs:
 build:
   machine: true
   steps:
     - checkout
     - run: |
        curl -L -o cirrus https://github.com/cirruslabs/cirrus-cli/releases/latest/download/cirrus-linux-amd64
        sudo mv cirrus /usr/local/bin/cirrus
        sudo chmod +x /usr/local/bin/cirrus
     - run: cirrus run
```

## TeamCity

Ensure that the CLI will run on the host itself (it should use a non-Dockerized agent) and this host has [Docker installed](https://docs.docker.com/engine/install/).

Create a build step with the "Command Line" runner type and the following custom script contents:

```
curl -L -o cirrus https://github.com/cirruslabs/cirrus-cli/releases/latest/download/cirrus-linux-amd64
chmod +x ./cirrus
./cirrus run
```

The resulting configuration should look like this:

![](images/teamcity-cirrus-run-build-step-ui.png)

**Note:** you can also preinstall the CLI on the agent itself to skip downloading it each time and just execute `cirrus run` during the step.

## Cloud Build

Here is an example of `cloudbuild.yaml` configuration file that runs Cirrus Tasks using CLI:

```yaml
steps:
  - name: 'cirrusci/cirrus-cli'
    args: ['run']
    env: ['CI=true']
```

If you want to use [Cloud Storage](https://cloud.google.com/storage) as a cache, Cirrus Labs provides a reference [HTTP proxy implementation](https://github.com/cirruslabs/google-storage-proxy) that transparently forwards all cache operations to the specified bucket.

Here's a modified version of the example above that stores cache entries in the bucket named `change-me`:

```yaml
steps:
  - name: 'docker'
    args:
      - 'run'
      - '-d'
      - '--name=gsp'
      - '--network=cloudbuild'
      - '--expose=80'
      - 'cirrusci/google-storage-proxy:latest'
      - '-address=0.0.0.0'
      - '-port=80'
      - '-bucket=change-me'
  - name: 'cirrusci/cirrus-cli'
    args: ['run', '--environment', 'CIRRUS_HTTP_CACHE_HOST=gsp']
    env: ['CI=true']
```

You can further configure [lifecycle rules](https://cloud.google.com/storage/docs/lifecycle) to automatically delete outdated cache objects.

## Cirrus CI

Cirrus CLI uses the same configuration format as [Cirrus CI](https://cirrus-ci.org/) and no additional configuration is required.
