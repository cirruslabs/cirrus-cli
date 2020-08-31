# Cirrus CLI

[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/cirrus-cli.svg?branch=master)](https://cirrus-ci.com/github/cirruslabs/cirrus-cli)

Cirrus CLI is a tool for running containerized tasks reproducibly in any environment. Run your tasks locally the same way
they are executed in CI or on your colleague's machine. Immutability of Docker containers ensures your project will compile
years from now regardless what versions of packages you'll have locally.

![Cirrus CLI Demo](images/cirrus-cli-demo.gif)

## Installation

* [Prebuilt Binary](intallation.md#prebuilt-binary)
* [Golang](intallation.md#golang)
* [Github Actions](intallation.md#github-actions)
* [Travis CI](intallation.md#travis-ci)
* [Circle CI](intallation.md#circle-ci)
* [Cirrus CI](intallation.md#cirrus-ci)

## Usage

Cirrus CLI reuses the [same YAML configuration format as Cirrus CI](https://cirrus-ci.org/guide/writing-tasks/) which allows to
reuse a large [list of examples](https://cirrus-ci.org/examples/) created by Cirrus CI community.

**Note:** Cirrus CLI can be used in any environment that has Docker installed. It can be your laptop or any CI system you already have
like Jenkins, Github Actions, Travis CI, etc. With Cirrus CLI it's no longer a requirement to use Cirrus CI in order to benefit from Cirrus
configuration format that we (Cirrus Labs) have crafted for so long and really proud of.

Here is an example of `.cirrus.yml` configuration file for testing a Go application with several Go versions:

```yaml
task:
  env:
    matrix:
      VERSION: 1.15
      VERSION: 1.14
  name: Tests (Go $VERSION)
  container:
    image: golang:$VERSION
  modules_cache:
    fingerprint_script: cat go.sum
    folder: $GOPATH/pkg/mod
  get_script: go get ./...
  build_script: go build ./...
  test_script: go test ./...
```

## Running Cirrus Tasks

To run Cirrus tasks, simply switch to a directory where the `.cirrus.yml` is located and run:
                                
```shell script
cirrus run
```

It is also possible to run a task by name:
                          
```shell script
cirrus run "Tests (Go 1.15)"
```

**Note:** Cirrus CLI only support [Linux `container`s](https://cirrus-ci.org/guide/linux/#linux-containers) instances at the moment
including [Dockerfile as a CI environment](https://cirrus-ci.org/guide/docker-builder-vm/#dockerfile-as-a-ci-environment) feature.

## Validating Cirrus Configuration

To validate a Cirrus configuration, simply switch to a directory where the `.cirrus.yml` is located and run:

```shell script
cirrus validate
```

## FAQ

<details>
 <summary>What is the relationship between Cirrus CI and Cirrus CLI?</summary>
 
 Cirrus CI was [released in the early 2018](https://medium.com/cirruslabs/introducing-cirrus-ci-a75cd1f49af0) with an idea
 to bring some innovation to CI space. A lot of things have changed in CI-as-a-service space since then but Cirrus CI
 pioneered many ideas in CI-as-a-service space including per-second billing and support for Linux, Windows and macOS all together.
 
 Over the past two and a half years we heard only positive feedback about Cirrus CI's YAML configuration format. Users liked how
 concise their configuration looked and that it was easy to reason about.
 
 Another feedback we heard from users was that it's hard to migrate from one CI to another. There is a need to rewrite CI configurations
 from one format into another that basically still locks into another vendor.
 
 With Cirrus CLI we are trying to solve the "vendor lock" problem by popularizing Cirrus configuration format and building
 community around it. Stay tuned for the upcoming option to use [Starlark templates instead of YAML](https://github.com/cirruslabs/cirrus-cli/issues/53)!
 
 Think of Cirrus CLI as a local executor of Cirrus Tasks on a single machine only in Docker containers and Cirrus CI as
 a remote executor of the same Cirrus Tasks in containers and VMs using a [variety of supported compute services](https://cirrus-ci.org/guide/supported-computing-services/)
 or using a [managed infrastructure with per-second billing](https://cirrus-ci.org/pricing/#compute-credits).
</details>
