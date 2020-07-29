# Cirrus CLI

[![Build Status](https://api.cirrus-ci.com/github/cirruslabs/cirrus-cli.svg)](https://cirrus-ci.com/github/cirruslabs/cirrus-cli)

CLI for executing Cirrus tasks locally.

## Installation

### Releases

Check the [releases page](https://github.com/cirruslabs/cirrus-cli/releases) for a pre-built `cirrus` binary for your platform.

### Go

If you have Go 1.14 installed, you can run:

```
go get github.com/cirruslabs/cirrus-cli/...
```

This will build and place the `cirrus` binary in `$GOPATH/bin`.

To be able to run `cirrus` command from anywhere, make sure that the `$GOPATH/bin` directory is added to your `PATH` environment variable (see [article in the Go wiki](https://github.com/golang/go/wiki/SettingGOPATH) for more details).

## Usage

### Validate

To validate a Cirrus CI configuration, simply switch to a directory where the `.cirrus.yml` is located and run:

```
cirrus validate
```

### Running

To run Cirrus CI tasks locally, simply switch to a directory where the `.cirrus.yml` is located and run:
                                
```
cirrus run
```

**Note:** Cirrus CLI only support [Linux `container`s](https://cirrus-ci.org/guide/linux/#linux-containers) instances at the moment.

