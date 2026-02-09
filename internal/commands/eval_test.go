package commands_test

import (
	"bytes"
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEvalTopLevelPrint(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`print("hello")`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalPrintlnAlias(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`
print("a", "b")
println("c", "d")
`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "a b\nc d\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalStdin(t *testing.T) {
	testutil.TempChdir(t)

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "-"}, `print("from-stdin")`)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "from-stdin\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalCirrusModules(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("data.json", []byte(`{"name":"cirrus","value":42}`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("script.star", []byte(`
load("cirrus", "fs", "json", "yaml")

doc = json.loads(fs.read("data.json"))
print(doc["name"])
print(yaml.dumps({"value": doc["value"]}).strip())
`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "cirrus\nvalue: 42\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalEnvironmentUsesProcessEnvironment(t *testing.T) {
	testutil.TempChdir(t)

	t.Setenv("CIRRUS_EVAL_TEST_ENV", "visible")

	if err := os.WriteFile("script.star", []byte(`
load("cirrus", "env")
print(env.get("CIRRUS_EVAL_TEST_ENV"))
`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "visible\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalPathsResolveFromCurrentDirectory(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.MkdirAll("scripts", 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("module.star", []byte(`
def message():
    return "from-root"
`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("scripts/module.star", []byte(`
def message():
    return "from-script-dir"
`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("payload.txt", []byte("root-file"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("scripts/payload.txt", []byte("script-file"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join("scripts", "main.star"), []byte(`
load("cirrus", "fs")
load("module.star", "message")

print(message())
print(fs.read("payload.txt"))
`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", filepath.Join("scripts", "main.star")}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "from-root\nroot-file\n", stdout)
	assert.Empty(t, stderr)
}

func TestEvalRuntimeErrorIncludesTraceback(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`
print("before")
a = []
print(a[0])
`), 0600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := evalCommandOutputs([]string{"eval", "script.star"}, "")
	if err == nil {
		t.Fatal("expected command to fail")
	}

	assert.Contains(t, stdout, "before\n")
	assert.Contains(t, stderr, "Traceback (most recent call last)")
}

func TestEvalHelpText(t *testing.T) {
	stdout, stderr, err := evalCommandOutputs([]string{"eval", "--help"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, stdout, "lightweight, LLM-friendly way to run Python-like scripts")
	assert.Contains(t, stdout, `load("cirrus", "http", "fs", "json", "yaml")`)
	assert.Contains(t, stdout, "print(...) and println(...)")
	assert.Contains(t, stdout, "cat script.star | cirrus eval -")
	assert.Contains(t, stdout, "cirrus eval scripts/task.star")
	assert.Contains(t, stdout, "https://www.githubstatus.com/api/v2/status.json")
	assert.Empty(t, stderr)
}

func TestEvalContextCancellation(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`
def burn():
    for i in range(0, 1000000000):
        pass

burn()
`), 0600); err != nil {
		t.Fatal(err)
	}

	command := commands.NewRootCmd()
	command.SetArgs([]string{"eval", "script.star"})

	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")
	command.SetOut(stdout)
	command.SetErr(stderr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- command.ExecuteContext(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected command to fail after context cancellation")
		}

		assert.Contains(t, err.Error(), "context canceled")
	case <-time.After(5 * time.Second):
		t.Fatal("command did not stop after context cancellation")
	}
}

func evalCommandOutputs(args []string, stdin string) (string, string, error) {
	command := commands.NewRootCmd()
	command.SetArgs(args)

	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetIn(bytes.NewBufferString(stdin))

	err := command.Execute()

	return stdout.String(), stderr.String(), err
}
