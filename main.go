//go:generate go run assets/generate/generate.go

package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/command"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/env"
	"github.com/avenga/couper/config/runtime"
)

var (
	fields = logrus.Fields{
		"type":    "couper_daemon",
		"build":   runtime.BuildName,
		"version": runtime.VersionName,
	}
	hook logrus.Hook
)

func main() {
	logrus.Exit(realmain(os.Args))
}

func realmain(arguments []string) int {
	args := command.NewArgs(arguments)

	if len(args) == 0 || command.NewCommand(args[0]) == nil {
		command.Help()
		return 1
	}

	cmd := args[0]
	args = args[1:]

	if cmd == "version" { // global options are not required, fast exit.
		_ = command.NewCommand(cmd).Execute(args, nil, nil)
		return 0
	}

	var filePath, logFormat string
	set := flag.NewFlagSet("global", flag.ContinueOnError)
	set.StringVar(&filePath, "f", config.DefaultFilename, "-f ./couper.hcl")
	set.StringVar(&logFormat, "log-format", config.DefaultSettings.LogFormat, "-log-format=common")
	err := set.Parse(args.Filter(set))
	if err != nil {
		newLogger(logFormat).WithFields(fields).Error(err)
		return 1
	}

	confFile, err := configload.LoadFile(filePath)
	if err != nil {
		newLogger(logFormat).WithFields(fields).Error(err)
		return 1
	}

	// The file gets initialized with the default settings, flag args are preferred over file settings.
	// Only override file settings if the flag value differ from the default.
	if logFormat != config.DefaultSettings.LogFormat {
		confFile.Settings.LogFormat = logFormat
	}
	logger := newLogger(confFile.Settings.LogFormat).WithFields(fields)

	wd, err := os.Getwd()
	if err != nil {
		logger.WithFields(fields).Error(err)
		return 1
	}
	logger.Infof("working directory: %s", wd)

	if err = command.NewCommand(cmd).Execute(args, confFile, logger); err != nil {
		logger.Error(err)
		return 1
	}
	return 0
}

// newLogger creates a log instance with the configured formatter.
// Since the format option may required to be correct in early states
// we parse the env configuration on every call.
func newLogger(format string) logrus.FieldLogger {
	logger := logrus.New()
	logger.Out = os.Stdout
	if hook != nil {
		logger.AddHook(hook)
		logger.Out = ioutil.Discard
	}

	settings := &config.Settings{LogFormat: format}
	env.Decode(settings)

	if settings.LogFormat == "json" {
		logger.Formatter = &logrus.JSONFormatter{FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "timestamp",
			logrus.FieldKeyMsg:  "message",
		}}
	}
	logger.Level = logrus.DebugLevel
	return logger
}
