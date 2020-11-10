package logs

import (
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/cirruslabs/echelon/renderers/config"
	"io"
	"os"
)

const (
	OutputAuto        = "auto"
	OutputInteractive = "interactive"
	OutputNoEmoji     = "noemoji"
	OutputSimple      = "simple"
	OutputTravis      = "travis"
	OutputGA          = "github-actions"
	OutputTeamCity    = "teamcity"
)

func DefaultFormat() string {
	return OutputAuto
}

func Formats() []string {
	return []string{
		OutputAuto,
		OutputInteractive,
		OutputNoEmoji,
		OutputSimple,
		OutputTravis,
		OutputGA,
		OutputTeamCity,
	}
}

func GetLogger(format string, verbose bool, logWriter io.Writer, logFile *os.File) (*echelon.Logger, func()) {
	if format == OutputAuto && envVariableIsTrue("TRAVIS") {
		format = OutputTravis
	}
	if format == OutputAuto && envVariableIsTrue("GITHUB_ACTIONS") {
		format = OutputGA
	}
	if format == OutputAuto && envVariableIsSet("TEAMCITY_VERSION") {
		format = OutputTeamCity
	}
	if format == OutputAuto && envVariableIsTrue("CI") {
		format = OutputSimple
	}
	if format == OutputAuto {
		format = OutputInteractive
	}

	var defaultSimpleRenderer = renderers.NewSimpleRenderer(logWriter, nil)
	var renderer echelon.LogRendered = defaultSimpleRenderer

	cancelFunc := func() {}

	switch format {
	case OutputInteractive:
		interactiveRenderer := renderers.NewInteractiveRenderer(logFile, nil)
		go interactiveRenderer.StartDrawing()
		cancelFunc = func() {
			interactiveRenderer.StopDrawing()
		}
		renderer = interactiveRenderer
	case OutputNoEmoji:
		interactiveRenderer := renderers.NewInteractiveRenderer(logFile, config.NewDefaultSymbolsOnlyRenderingConfig())
		go interactiveRenderer.StartDrawing()
		cancelFunc = func() {
			interactiveRenderer.StopDrawing()
		}
		renderer = interactiveRenderer
	case OutputTravis:
		renderer = NewTravisCILogsRenderer(defaultSimpleRenderer)
	case OutputGA:
		renderer = NewGithubActionsLogsRenderer(defaultSimpleRenderer)
	case OutputTeamCity:
		renderer = NewTeamCityLogsRenderer(defaultSimpleRenderer)
	}

	logger := echelon.NewLogger(echelon.InfoLevel, renderer)

	if verbose {
		logger = echelon.NewLogger(echelon.DebugLevel, renderer)
	}

	return logger, cancelFunc
}

func envVariableIsSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

func envVariableIsTrue(name string) bool {
	return os.Getenv(name) == "true"
}
