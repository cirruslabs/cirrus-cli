# Starlark templating guide

If you've already read the [Starlark guide](STARLARK.md) — then you're one step away from making your own template that you and other people can re-use.

If not, then we highly recommend that you do so — it's just a stripped-down Python after all.

## Quick start

The most straightforward way to start is to [use our example template](https://github.com/cirrus-templates/example).

Assuming that you've instantiated the template into `github.com/user/dynamic-template`, you can now `load()` it from anywhere like this:

```python
load("github.com/user/dynamic-template", "hello_world")
```

Proceed with modifying the `lib.star` to your own liking, additionally splitting it into multiple files by using [local module loading](STARLARK.md#local).

And don't forget to add a `cirrus-template` topic to make your template discoverable!

## Differences with Starlark configurations

The treatment of Starlark templates is mostly similar to `.cirrus.star`, but there are some differences that apply only to Starlark templates.

### Entrypoint

The most significant difference is that templates can be loaded without specifying the name of a `.star`-file:

```python
load("github.com/cirrus-templates/golang", "detect_tasks")
```

When no `.star` file to load is specified, the convention is to load `lib.star` by default. So, behind the scenes this will be expanded into:

```python
load("github.com/cirrus-templates/golang/lib.star", "detect_tasks")
```

## Testing

If your template generates tasks, you can test it's expected output by creating a directory anywhere in your project and placing a `.cirrus.expected.yml` file there.

You'll also need to place a `.cirrus.star` file there, which call your template functions that you want to test.

Once everything is set-up, run the following CLI command from your project's root:

```
cirrus internal test
```

This CLI command will find all directories with `.cirrus.expected.yml` file in them, run the `.cirrus.star` from the same directory and compare the results.
