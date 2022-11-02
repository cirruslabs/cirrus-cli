# Starlark module guide

If you've already read the [Starlark guide](https://cirrus-ci.org/guide/programming-tasks/) — then you're one step away from making your own module that you and other people can re-use.

If not, then we highly recommend that you do so — it's just a stripped-down Python after all.

## Quick start

The most straightforward way to start is to [use our example module](https://github.com/cirrus-modules/example).

Assuming that you've instantiated the module into `github.com/user/dynamic-module`, you can now `load()` it from anywhere like this:

```python
load("github.com/user/dynamic-module", "hello_world")
```

Proceed with modifying the `lib.star` to your own liking, additionally splitting it into multiple files by using [local module loading](https://cirrus-ci.org/guide/programming-tasks/#local).

And don't forget to add a `cirrus-module` topic to make your module discoverable!

## Examples

There are couple of module examples available under [`cirrus-modules`](https://github.com/cirrus-modules) organization:

* [`helpers`](https://github.com/cirrus-modules/helpers) - a set of helper functions to build Cirrus tasks from Starlark.
* [`golang`](https://github.com/cirrus-modules/golang) - module to auto-configure tasks for a Go repository.

## Differences with Starlark configurations

The treatment of Starlark modules is mostly similar to `.cirrus.star`, but there are some differences that apply only to Starlark modules.

### Entrypoint

The most significant difference is that modules can be loaded without specifying the name of a `.star`-file:

```python
load("github.com/cirrus-modules/golang", "detect_tasks")
```

When no `.star` file to load is specified, the convention is to load `lib.star` by default. So, behind the scenes this will be expanded into:

```python
load("github.com/cirrus-modules/golang/lib.star", "detect_tasks")
```

## Testing

If your module generates tasks, you can test it's expected output by creating a directory anywhere in your project and placing a `.cirrus.expected.yml` file there.

You'll also need to place there a `.cirrus.star` file which loads your module functions you want to test.

It's also possible to verify logs produced in the process of executing your module by creating `.cirrus.expected.log` file with the expected logs.

Once everything is set-up, run the following CLI command from your project's root:

```
cirrus internal test
```

This CLI command will find all directories with `.cirrus.expected.yml` file in them, run the `.cirrus.star` from the same directory and compare the results with the expected `.cirrus.expected.yml`.

### Test configuration file

Some Starlark modules use the [`env` dict](https://cirrus-ci.org/guide/programming-tasks/#env) which contents depends on the environment.

To mock the contents of this dict, you can either specify a `-e` flag to `cirrus internal test` (see [testing private repositories](#testing-private-repositories) below), or create the following `.cirrus.testconfig.yml` in the test's directory:

```yaml
env:
  CIRRUS_TAG: "v0.1.0"
```

Similarly, to mock the [`changes_include()`](https://cirrus-ci.org/guide/programming-tasks/#changes_include) or [`changes_include_only()`](https://cirrus-ci.org/guide/programming-tasks/#changes_include_only) functions behavior, specify the files that were affected:

```yaml
affected_files:
  - ci/build.sh
```

The resulting file will look like this:

```yaml
env:
  CIRRUS_TAG: "v0.1.0"

affected_files:
  - ci/build.sh
```

These two additions combined will ensure that when the test runs:

* `env.get("CIRRUS_TAG")` will return `v0.1.0`
* `changes_include("**.sh")` will return `True`

### Testing private repositories

To aid in testing private repositories that require an authentication token, `cirrus internal test` supports specifying additional environment variables from command-line.

All you need is a [personal access token](https://github.com/settings/tokens?type=beta) that has access to the repository that you're going to `load(...)`, and once you get one, specify it as a value to `CIRRUS_REPO_CLONE_TOKEN` environment variable:

```
cirrus internal test -e CIRRUS_REPO_CLONE_TOKEN=<GitHub personal access token>
```

You can also specify this environment variable in the `.cirrus.testconfig.yml` to achieve the same effect.
