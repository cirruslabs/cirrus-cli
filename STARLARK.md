# Starlark

In addition to the [YAML configuration format](https://cirrus-ci.org/guide/writing-tasks/), Cirrus CLI supports evaluation of `.cirrus.star` scripts written in the Starlark language. 

[Starlark](https://github.com/bazelbuild/starlark) is essentially a stripped-down dialect of Python. The major differences are explained in the [Bazel documentation](https://docs.bazel.build/versions/master/skylark/language.html).

Cirrus CLI embeds a Starlark interpreter and introduces the following additions:

* [entrypoints](#entrypoints)
* [module loading](#module-loading)
* [builtins](#builtins)

## Compatibility with the YAML format

When you execute `cirrus run`, it looks for the following files in the current directory:

* `.cirrus.yml`
* `.cirrus.star`

While the first configuration is sent directly to the YAML parser, the second configuration is evaluated with the Starlark interpreter first, and only then the evaluation output is parsed as YAML.

You can also have both `.cirrus.yml` and `.cirrus.star` configurations at the same time — their output will be simply merged.

## Writing Starlark scripts

One of the most trivial `.cirrus.star` examples look like this:

```python
def main(ctx):
    return [
        {
            "container": {
                "image": "debian:latest",
            },
            "script": "make",
        },
    ]
```

Once `run` in the Cirrus CLI, it will internally generate and parse the following YAML configuration to produce the actual tasks to run:

```yaml
task:
    container:
      image: debian:latest
    script: make
```

You might ask why not simply use the YAML format here? With Starlark, you can generate parts of the configuration dynamically based on some external conditions: by [making an HTTP request](#http) to check the previous build status or by [parsing files inside the repository](#fs) to pick up some common settings (for example, parse `package.json` to see if it contains `lint` script and generate a linting task).
 
And even more importantly: with the [module loading](#module-loading) you can re-use other people's code to avoid wasting time on things written from scratch. For example, there are official [task helpers](https://github.com/cirrus-templates/helpers) available that reduce the boilerplate when generating tasks:

```python
load("github.com/cirrus-templates/helpers", "task", "container", "script")

def main(ctx):
    return [
        task(instance=container("debian:latest"), instructions=[script("make")]),
    ]
```

## Entrypoints

Different events will call different top-level functions in the `.cirrus.star`. These functions reserve certain names and will be called with different arguments depending on an event which triggered the execution.

Currently only build generation entrypoint is supported with more to come in the future. For example, there will be a way to declare a function to be called on a task failure to analyze logs and if necessary re-run the task automatically.

### Build generation

Entrypoint: `main(ctx)`

Cirrus CLI will call this function in the `.cirrus.star` when being executed as `cirrus run` to retrieve a list of tasks to run.

Arguments:

* `ctx` — reserved for future use

Return value:

* a list of dicts, where each dict closely represents a task in the YAML configuration format

## Module loading

Module loading is done through the Starlark's [`load()`](https://github.com/bazelbuild/starlark/blob/master/spec.md#load-statements) statement.

Besides the ability to load [builtins](#builtins) with it, Cirrus CLI can load other `.star` files from local and remote locations to facilitate code re-use.

### Local

Local loads are relative to the project's root (where `.cirrus.star` is located):

```python
load(".ci/notify-slack.star", "notify_slack")
```

### Remote from Git

To load a specific branch of the template from GitHub:

```python
load("github.com/cirrus-templates/golang@master", "task", "container")
```

In the example above, the name of the `.star` file was not provided, because `lib.star` is assumed by default. This is equivalent to:

```python
load("github.com/cirrus-templates/golang/lib.star@master", "task", "container")
```

You can also specify an exact commit hash instead of the `master` branch name to prevent accidental changes.

To load `.star` files from repositories other than GitHub, add a `.git` suffix at the end of the repository name, for example:

```python
load("gitlab.com/fictional/repository.git/validator.star", "validate")
                                     ^^^^ note the suffix
```

## Builtins

Cirrus CLI provides builtins all nested in the `cirrus` module that greatly extend what can be done with the Starlark alone.

### `fs`

These builtins allow for read-only filesystem access.

All paths are relative to the project's directory.

#### `fs.exists(path)`

Returns `True` if `path` exists and `False` otherwise.

#### `fs.read(path)`

Returns a [`string`](https://github.com/bazelbuild/starlark/blob/master/spec.md#strings) with the file contents or `None` if the file doesn't exist.

Note that this is an error to read a directory with `fs.read()`.

#### `fs.readdir(dirpath)`

Returns a [`list`](https://github.com/bazelbuild/starlark/blob/master/spec.md#lists) of [`string`'s](https://github.com/bazelbuild/starlark/blob/master/spec.md#strings) with names of the entries in the directory.

Note that this is an error to read a file with `fs.readdir()`.

Example:

```python
load("cirrus", "fs")

def main(ctx):
    tasks = base_tasks()

    if fs.exists("go.mod"):
        tasks += go_tasks()

    return tasks
```

### `env`

While not technically a builtin, `env` is dict that contains environment variables passed via `cirrus run --environment`.

Example:

```python
load("cirrus", "env")

def main(ctx):
    tasks = base_tasks()

    if env.get("CIRRUS_TAG") != None:
        tasks += release_tasks()

    return tasks
```

### `changes_include`

`changes_include()` is a Starlark's alternative to the [changesInclude()](https://cirrus-ci.org/guide/writing-tasks/#supported-functions) function commonly found in the YAML configuration files.

It takes at least one [`string`](https://github.com/bazelbuild/starlark/blob/master/spec.md#strings) with a pattern and returns a [`bool`](https://github.com/bazelbuild/starlark/blob/master/spec.md#booleans) that represents whether any of the specified patterns matched any of the affected files in the running context.

Currently supported contexts:

* [`main()` entrypoint](#build-generation)

Example:

```python
load("cirrus", "changes_include")

def main(ctx):
    tasks = base_tasks()

    if changes_include("Dockerfile"):
        tasks += docker_task()

    return tasks
```

### `http`

Provides HTTP client implementation with `http.get()`, `http.post()` and other HTTP method functions.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/http) for more details.

### `hash`

Provides cryptographic hashing functions, such as `hash.md5()`, `hash.sha1()` and `hash.sha256()`.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/hash) for more details.

### `base64`

Provides Base64 encoding and decoding functions using `base64.encode()` and `base64.decode()`.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/encoding/base64) for more details.

### `json`

Provides JSON document marshalling and unmarshalling using `json.dumps()` and `json.loads()` functions.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/encoding/json) for more details.

### `yaml`

Provides YAML document marshalling and unmarshalling using `yaml.dumps()` and `yaml.loads()` functions.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/encoding/yaml) for more details.

### `re`

Provides regular expression functions, such as `findall()`, `split()` and `sub()`.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/re) for more details.

### `zipfile`

`cirrus.zipfile` module provides methods to read Zip archives.

You instantiate a `ZipFile` object using `zipfile.ZipFile(data)` function call and then call `namelist()` and `open(filename)` methods to retrieve information about archive contents.

Refer to the [starlib's documentation](https://github.com/qri-io/starlib/tree/master/zipfile) for more details.

Example:

```python
load("cirrus", "fs", "zipfile")

def is_java_archive(path):
    # Read Zip archive contents from the filesystem
    archive_contents = fs.read(path)
    if archive_contents == None:
        return False

    # Open Zip archive and a file inside of it
    zf = zipfile.ZipFile(archive_contents)
    manifest = zf.open("META-INF/MANIFEST.MF")

    # Does the manifest contain the expected version?
    if "Manifest-Version: 1.0" in manifest.read():
        return True

    return False
```

## Security

### Remote loads

Cirrus CLI always uses HTTPS to fetch files from Git.

### Builtins

While builtins provide functionality that is considered non-altering to the local system, there are some cases when this may not be enough:

* `cirrus.http` methods can access local services running on `127.0.0.1` or inside of LAN and potentially interact with the services running on these hosts in malicious ways

It's recommended that you don't run Starlark scripts from potentially untrusted sources, similarly to how you probably wouldn't run build scripts from random repositories found on the internet.
