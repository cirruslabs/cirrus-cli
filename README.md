# Cirrus CLI

[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/cirrus-cli.svg)](https://cirrus-ci.com/github/cirruslabs/cirrus-cli)

Cirrus CLI is a tool for running containerized tasks reproducibly in any environment. Run your tasks locally the same way
they are executed in CI or on your colleague's computer. Immutability of Docker containers ensures your project will compile
years form now regarding what versions of packages you have locally.

![Cirrus CLI Demo](images/cirrus-cli-demo.gif)

**Note:** Even though Cirrus CLI is using the same configuration format as [Cirrus CI](https://cirrus-ci.org/), 
Cirrus CLI can be used in any environment that has Docker installed including any CI system like Jenkins, Github Actions, etc.
Think of Cirrus CLI as an executor of Cirrus tasks on a single machine and Cirrus CI as a cloud executor.

## Installation

### Releases

Check the [releases page](https://github.com/cirruslabs/cirrus-cli/releases) for a pre-built `cirrus` binary for your platform.

### Go

If you have [Go](https://golang.org/) installed, you can run:

```
go get github.com/cirruslabs/cirrus-cli/...
```

This will build and place the `cirrus` binary in `$GOPATH/bin`.

To be able to run `cirrus` command from anywhere, make sure the `$GOPATH/bin` directory is added to your `PATH`
environment variable (see [article in the Go wiki](https://github.com/golang/go/wiki/SettingGOPATH) for more details).

## Usage

### Configuration Format

Cirrus CLI uses the [same configuration format as Cirrus CI](https://cirrus-ci.org/guide/writing-tasks/) which allows to
reuse a large [list of ready-to-use examples](https://cirrus-ci.org/examples/) created by Cirrus CI community.

### Validate

To validate a Cirrus configuration, simply switch to a directory where the `.cirrus.yml` is located and run:

```
cirrus validate
```

### Running

To run Cirrus tasks, simply switch to a directory where the `.cirrus.yml` is located and run:
                                
```
cirrus run
```

It is also possible to run a task by name:
                          
```
cirrus run "Tests (Go 1.15)"
```

**Note:** Cirrus CLI only support [Linux `container`s](https://cirrus-ci.org/guide/linux/#linux-containers) instances at the moment.

