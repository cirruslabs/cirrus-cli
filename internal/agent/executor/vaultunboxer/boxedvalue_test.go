package vaultunboxer_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/vaultunboxer"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNonBoxedValues(t *testing.T) {
	// Empty value
	_, err := vaultunboxer.NewBoxedValue("")
	require.ErrorIs(t, err, vaultunboxer.ErrNotABoxedValue)

	// Unterminated Vault-boxed value
	_, err = vaultunboxer.NewBoxedValue("VAULT[")
	require.ErrorIs(t, err, vaultunboxer.ErrNotABoxedValue)
}

func TestInvalidBoxedValues(t *testing.T) {
	// Empty value
	_, err := vaultunboxer.NewBoxedValue("VAULT[]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value with not enough arguments
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value with invalid argument
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.path extraneous]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value that contains a selector with empty elements
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value that contains a argument but is missing a selector
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path arg=value]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value that contains a argument with empty value
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.path arg=]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)

	// Value that contains a argument with empty key
	_, err = vaultunboxer.NewBoxedValue("VAULT[some/path some.path =value]")
	require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)
}

func TestArguments(t *testing.T) {
	trials := []struct {
		Name          string
		RawBoxedValue string
		Expected      map[string][]string
	}{
		{
			Name:          "no arguments",
			RawBoxedValue: "VAULT[some/path some.path]",
			Expected:      map[string][]string{},
		},
		{
			Name:          "one argument",
			RawBoxedValue: "VAULT[some/path some.path arg=value]",
			Expected: map[string][]string{
				"arg": {"value"},
			},
		},
		{
			Name:          "multiple arguments",
			RawBoxedValue: "VAULT[some/path some.path arg=value arg2=value2]",
			Expected: map[string][]string{
				"arg":  {"value"},
				"arg2": {"value2"},
			},
		},
		{
			Name:          "multiple arguments with the same name",
			RawBoxedValue: "VAULT[some/path some.path arg=value arg=value2]",
			Expected: map[string][]string{
				"arg": {"value", "value2"},
			},
		},
	}

	for _, trial := range trials {
		t.Run(trial.Name, func(t *testing.T) {
			boxedValue, err := vaultunboxer.NewBoxedValue(trial.RawBoxedValue)
			require.NoError(t, err)

			require.Equal(t, trial.Expected, boxedValue.VaultPathArgs())
		})
	}
}

func TestSelectorInvalidCombinations(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"first_key": "first secret key value",
			"integer":   42,
		},
	}

	trials := []struct {
		Name          string
		RawBoxedValue string
	}{
		{
			Name:          "querying elements in a scalar element",
			RawBoxedValue: "VAULT[secret/data/keys data.first_key.not_in_dict]",
		},
		{
			Name:          "querying a non-existent element",
			RawBoxedValue: "VAULT[secret/data/keys data.nonexistent]",
		},
		{
			Name:          "when querying terminating element that is not a string and not a dictionary",
			RawBoxedValue: "VAULT[secret/data/keys data.integer]",
		},
	}

	for _, trial := range trials {
		t.Run(trial.Name, func(t *testing.T) {
			selector, err := vaultunboxer.NewBoxedValue(trial.RawBoxedValue)
			require.NoError(t, err)

			_, err = selector.Select(data)
			require.ErrorIs(t, err, vaultunboxer.ErrInvalidBoxedValue)
		})
	}
}

func TestSelector(t *testing.T) {
	const (
		firstSecretKeyValue  = "first secret key value"
		secondSecretKeyValue = "second secret key value"
	)

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"first_key": firstSecretKeyValue,
			"extra": map[string]interface{}{
				"second_key": secondSecretKeyValue,
			},
		},
	}

	trials := []struct {
		Name          string
		RawBoxedValue string
		Expected      string
	}{
		{
			Name:          "first key",
			RawBoxedValue: "VAULT[secret/data/keys data.first_key]",
			Expected:      firstSecretKeyValue,
		},
		{
			Name:          "second key",
			RawBoxedValue: "VAULT[secret/data/keys data.extra.second_key]",
			Expected:      secondSecretKeyValue,
		},
	}

	for _, trial := range trials {
		t.Run(trial.Name, func(t *testing.T) {
			selector, err := vaultunboxer.NewBoxedValue(trial.RawBoxedValue)
			require.NoError(t, err)

			result, err := selector.Select(data)
			require.NoError(t, err)
			require.Equal(t, trial.Expected, result)
		})
	}
}

func TestWithAndWithoutCache(t *testing.T) {
	valueThatUsesCache, err := vaultunboxer.NewBoxedValue("VAULT[path key]")
	require.NoError(t, err)
	require.True(t, valueThatUsesCache.UseCache())

	valueThatDoesNotUseCache, err := vaultunboxer.NewBoxedValue("VAULT_NOCACHE[path key]")
	require.NoError(t, err)
	require.False(t, valueThatDoesNotUseCache.UseCache())
}
