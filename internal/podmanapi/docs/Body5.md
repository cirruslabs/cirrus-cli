# Body5

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AttachStderr** | **bool** | Attach to stderr of the exec command | [optional] [default to null]
**AttachStdin** | **bool** | Attach to stdin of the exec command | [optional] [default to null]
**AttachStdout** | **bool** | Attach to stdout of the exec command | [optional] [default to null]
**Cmd** | **[]string** | Command to run, as a string or array of strings. | [optional] [default to null]
**DetachKeys** | **string** | \&quot;Override the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-&lt;value&gt; where &lt;value&gt; is one of: a-z, @, ^, [, , or _.\&quot;  | [optional] [default to null]
**Env** | **[]string** | A list of environment variables in the form [\&quot;VAR&#x3D;value\&quot;, ...] | [optional] [default to null]
**Privileged** | **bool** | Runs the exec process with extended privileges | [optional] [default to false]
**Tty** | **bool** | Allocate a pseudo-TTY | [optional] [default to null]
**User** | **string** | \&quot;The user, and optionally, group to run the exec process inside the container. Format is one of: user, user:group, uid, or uid:gid.\&quot;  | [optional] [default to null]
**WorkingDir** | **string** | The working directory for the exec process inside the container. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

