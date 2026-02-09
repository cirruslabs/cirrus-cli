package commands_test

import (
	"bytes"
	"github.com/cirruslabs/cirrus-cli/internal/commands"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestEvalTopLevelPrint(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`print("hello")`), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := evalCommandOutput([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello\n", output)
}

func TestEvalPrintlnAlias(t *testing.T) {
	testutil.TempChdir(t)

	if err := os.WriteFile("script.star", []byte(`
print("a", "b")
println("c", "d")
`), 0600); err != nil {
		t.Fatal(err)
	}

	output, err := evalCommandOutput([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "a b\nc d\n", output)
}

func TestEvalStdin(t *testing.T) {
	testutil.TempChdir(t)

	output, err := evalCommandOutput([]string{"eval", "-"}, `print("from-stdin")`)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "from-stdin\n", output)
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

	output, err := evalCommandOutput([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "cirrus\nvalue: 42\n", output)
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

	output, err := evalCommandOutput([]string{"eval", "script.star"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "visible\n", output)
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

	output, err := evalCommandOutput([]string{"eval", filepath.Join("scripts", "main.star")}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "from-root\nroot-file\n", output)
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

	output, err := evalCommandOutput([]string{"eval", "script.star"}, "")
	if err == nil {
		t.Fatal("expected command to fail")
	}

	assert.Contains(t, output, "before\n")
	assert.Contains(t, output, "Traceback (most recent call last)")
}

func TestEvalHelpText(t *testing.T) {
	output, err := evalCommandOutput([]string{"eval", "--help"}, "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output, "lightweight, LLM-friendly way to run Python-like scripts")
	assert.Contains(t, output, `load("cirrus", "http", "fs", "json", "yaml")`)
	assert.Contains(t, output, "print(...) and println(...)")
	assert.Contains(t, output, "cat script.star | cirrus eval -")
	assert.Contains(t, output, "cirrus eval scripts/task.star")
	assert.Contains(t, output, "https://www.githubstatus.com/api/v2/status.json")
}

func evalCommandOutput(args []string, stdin string) (string, error) {
	command := commands.NewRootCmd()
	command.SetArgs(args)

	output := bytes.NewBufferString("")
	command.SetOut(output)
	command.SetErr(output)
	command.SetIn(bytes.NewBufferString(stdin))

	err := command.Execute()

	return output.String(), err
}
