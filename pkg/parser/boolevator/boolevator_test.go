package boolevator_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/boolevator"
	"github.com/stretchr/testify/assert"
	"testing"
)

func evalHelper(t *testing.T, expr string, env map[string]string) bool {
	evaluation, err := boolevator.New().Eval(expr, env)
	if err != nil {
		t.Fatal(err)
	}

	return evaluation
}

func TestSimple(t *testing.T) {
	assert.True(t, evalHelper(t, "'Foo' == 'Foo'", nil))
	assert.False(t, evalHelper(t, "'Foo' == 'Bar'", nil))
}

func TestEmptyVariables(t *testing.T) {
	assert.True(t, evalHelper(t, "$FOO == ''", nil))
}

func TestSimpleParentesis(t *testing.T) {
	env := map[string]string{
		"CIRRUS_USER_COLLABORATOR": "true",
	}

	assert.True(t, evalHelper(t, "($CIRRUS_PR == '' || $CIRRUS_TAG != '') && $CIRRUS_USER_COLLABORATOR == 'true'", env))
}

func TestSimpleVariables(t *testing.T) {
	env := map[string]string{
		"FOO": "Foo",
		"BAR": "Bar",
	}

	assert.True(t, evalHelper(t, "'Foo' == $FOO", env))
	assert.False(t, evalHelper(t, "'Foo' == $BAR", env))
}

func TestSingleVariables(t *testing.T) {
	env := map[string]string{
		"FOO": "true",
	}

	assert.True(t, evalHelper(t, "$FOO", env))
}

func TestComplex(t *testing.T) {
	assert.False(t, evalHelper(t, "!('Foo' == 'Foo')", nil))
	assert.True(t, evalHelper(t, "'Foo' == 'Bar' || 'Foo' != 'Bar'", nil))
}

func TestRegEx(t *testing.T) {
	assert.True(t, evalHelper(t, "'release-.*' =~ 'release-2018.1'", nil))
	assert.False(t, evalHelper(t, "'release-.*' !=~ 'release-2018.1'", nil))
	assert.True(t, evalHelper(t, "'release-.*' !=~ 'foo'", nil))
	assert.True(t, evalHelper(t, "'1.2.34' =~ '\\d+\\.\\d+\\.\\d+'", nil))
}

func TestRegExMultiline(t *testing.T) {
	changelog := `[tests] added tests
[documentation] documented API
[fixes] fixed bugs
`

	assert.True(t, evalHelper(t, "$CHANGELOG =~ '.*\\[documentation\\].*'",
		map[string]string{"CHANGELOG": changelog}))
}

func TestQuotes(t *testing.T) {
	assert.True(t, evalHelper(t, "$CIRRUS_CHANGE_MESSAGE == \"test 'foo'\"", map[string]string{"CIRRUS_CHANGE_MESSAGE": "test 'foo'"}))
	assert.True(t, evalHelper(t, "$CIRRUS_CHANGE_MESSAGE =~ \".*test 'foo'.*\"", map[string]string{"CIRRUS_CHANGE_MESSAGE": "test 'foo'"}))
}

func TestPrRegEx(t *testing.T) {
	assert.True(t, evalHelper(t, "'pull/.*' =~ 'pull/123'", nil))
	assert.True(t, evalHelper(t, "'pull/123' =~ 'pull/.*'", nil))
	assert.True(t, evalHelper(t, "'branch-foo' =~ 'branch-.*'", nil))
}

func TestTagRegEx(t *testing.T) {
	assert.True(t, evalHelper(t, "'v2.0' =~ 'v\\d+\\.\\d+.*'", nil))
}

func TestRegExNoFailFast(t *testing.T) {
	assert.False(t, evalHelper(t, "'not a regexp :)' =~ '.*actual regexp.*'", nil))
	assert.True(t, evalHelper(t, "'not a regexp :)' !=~ '.*actual regexp.*'", nil))
}

func TestIn(t *testing.T) {
	assert.True(t, evalHelper(t, "'pull' in 'pull/123'", nil))
	assert.False(t, evalHelper(t, "'pull' in 'pr/123'", nil))

	assert.True(t, evalHelper(t, "$CIRRUS_BRANCH == 'integration'",
		map[string]string{"CIRRUS_BRANCH": "integration"}))
}

func TestFunction(t *testing.T) {
	functions := map[string]boolevator.Function{
		"changesInclude": func(arguments ...interface{}) interface{} {
			return "true"
		},
	}

	evaluation, err := boolevator.New(boolevator.WithFunctions(functions)).Eval("changesInclude('*.txt')", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, evaluation)
}

func TestLongExpression(t *testing.T) {
	env := map[string]string{
		"CIRRUS_USER_COLLABORATOR": "false",
		"CIRRUS_BRANCH":            "trying",
	}

	assert.True(t, evalHelper(t, "$CIRRUS_USER_COLLABORATOR == 'true' || $CIRRUS_BRANCH == 'master' || "+
		"$CIRRUS_BRANCH == 'staging' || $CIRRUS_BRANCH == 'trying'", env))
}

func TestManyIfs(t *testing.T) {
	env := map[string]string{"CIRRUS_BRANCH": "master"}

	assert.False(t, evalHelper(t, "$CIRRUS_BASE_BRANCH == 'dev' || $CIRRUS_BRANCH == 'dev' || "+
		"$CIRRUS_BASE_BRANCH == 'beta' || $CIRRUS_BRANCH == 'beta' || $CIRRUS_BASE_BRANCH == 'stable' || "+
		"$CIRRUS_BRANCH == 'stable'", env),
	)
}

func TestComplexTag(t *testing.T) {
	env := map[string]string{"CIRRUS_TAG": "v1.2.3-rc1"}

	assert.True(t, evalHelper(t, "$CIRRUS_TAG =~ 'v\\d+(\\.\\d+){2}(-.*)?'", env))
}
