package suw

import (
	"errors"

	"github.com/stephen-fox/versionutil"
)

const (
	DefaultExecutablePath = executableParentPath + executableName

	ErrorNoSuchUpdate             = "The specified update does not exist"
	ErrorUpdatesServerUnreachable = "The Apple updates server is unreachable"

	executableParentPath  = "/usr/sbin/"
	executableName        = "softwareupdate"

	updatePrefix        = "*"
	updateDetailsPrefix = "\t"
	progressPrefix      = "Progress: "
	progressSuffix      = "%"

	updateSizeSuffix    = "K"
	restartRequired     = "[restart]"
	noSuchUpdateSuffix  = "No such update"

	listUpdatesArg   = "-l"
	installUpdateArg = "-i"
	verboseArg       = "--verbose"
)

type Update struct {
	Name            string
	ApplicationName string
	Version         versionutil.Version
	SizeMegabytes   uint64
	IsRestartNeeded bool
}

func (o Update) HasUpdateSize() bool {
	return o.SizeMegabytes > 0
}

var (
	TargetCliApi CliApi = GetDefaultCliApi()
)

// GetUpdates gets all available updates.
func GetUpdates() ([]Update, error) {
	output, err := TargetCliApi.Execute(listUpdatesArg)
	if err != nil {
		return []Update{}, err
	}

	var updates []Update

	for i, l := range output {
		nextLine := ""

		if i < len(output) - 1 {
			nextLine = output[i+1]
		}

		isUpdate, update := TargetCliApi.IsUpdate(l, nextLine)
		if isUpdate {
			updates = append(updates, update)
		}
	}

	return updates, nil
}

// InstallUpdates installs an update.
func InstallUpdate(updateName string) error {
	return InstallUpdateVerbose(updateName, nil)
}

// InstallUpdateVerbose installs an update and provides installation  progress
// percentages to the specified channel.
func InstallUpdateVerbose(updateName string, progressPercentages chan int) error {
	outputs := make(chan string)

	var noSuchUpdateErr error

	go func() {
		for line := range outputs {
			if TargetCliApi.IsNoSuchUpdate(updateName, line) {
				noSuchUpdateErr = errors.New(ErrorNoSuchUpdate)
			}

			if progressPercentages != nil {
				isProgress, percent := TargetCliApi.IsInstallProgress(line)
				if isProgress {
					progressPercentages <- percent
				}
			}
		}
	}()

	err := TargetCliApi.ExecuteToChan(outputs, verboseArg, installUpdateArg, updateName)
	if err != nil {
		return err
	}

	if noSuchUpdateErr != nil {
		return noSuchUpdateErr
	}

	return nil
}