package macosupdateutil

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/stephen-fox/versionutil"
)

type CliApi interface {
	// SetExecutablePath sets the path to the softwareupdate executable.
	SetExecutablePath(path string)

	// Execute executes softwareupdate.
	Execute(args ...string) (output []string, err error)

	// ExecuteToChan executes softewareupdate and redirects its combined
	// output to the specified channel.
	ExecuteToChan(outputs chan string, args ...string) error

	// GetApplicationName gets the application affected by an updated
	// from softwareupdate output.
	GetApplicationName(line string) string

	// IsUpdate checks if the output from softwareupdate is a line, and
	// returns an Update representing it.
	IsUpdate(line string, nextLine string) (bool, Update)

	// GetUpdateSizeMegabytes gets the update's size in megabytes by parsing
	// the output of softwareupdate.
	GetUpdateSizeMegabytes(line string) (uint64, error)

	// IsInstallProgress checks if the output is an installation progress
	// update and returns the percentage associated with the progress.
	IsInstallProgress(line string) (bool, int)

	// IsNoSuchUpdate returns true if the softwareupdate output states that
	// no such update exists.
	IsNoSuchUpdate(updateName string, line string) bool
}

type defaultCliApi struct {
	ExecutablePath string
}

func (o *defaultCliApi) SetExecutablePath(path string) {
	o.ExecutablePath = path
}

func (o *defaultCliApi) Execute(args ...string) (output []string, err error) {
	outputs := make(chan string)
	var lines []string
	listen := true

	go func() {
		for listen {
			line := <- outputs
			lines = append(lines, line)
		}
	}()

	err = o.ExecuteToChan(outputs, args...)

	listen = false

	if err != nil {
		return []string{}, err
	}

	return lines, nil
}

func (o *defaultCliApi) ExecuteToChan(outputs chan string, args ...string) error {
	command := exec.Command(o.ExecutablePath, args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	combinedOutput := io.MultiReader(stdout, stderr)

	err = command.Start()
	if err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(combinedOutput)
		lastLine := ""

		for scanner.Scan() {
			line := scanner.Text()

			if line == lastLine {
				continue
			}

			lastLine = line

			outputs <- line
		}
	}()

	err = command.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (o *defaultCliApi) IsUpdate(line string, nextLine string) (bool, Update) {
	// We cannot remove trailing spaces because softwareupdate
	// treats them as part of the update name.
	line = strings.TrimLeft(line, " ")

	if !strings.HasPrefix(line, updatePrefix) {
		return false, Update{}
	}

	refName := strings.TrimPrefix(line, updatePrefix+" ")
	appName := o.GetApplicationName(refName)

	update := Update{
		Name:            refName,
		ApplicationName: appName,
	}

	v, versionErr := versionutil.StringToVersion(update.Name)
	if versionErr == nil {
		update.Version = v
	}

	if nextLine != "" && strings.HasPrefix(nextLine, updateDetailsPrefix) {
		mb, sizeErr := o.GetUpdateSizeMegabytes(nextLine)
		if sizeErr == nil {
			update.SizeMegabytes = mb
		}

		if !update.Version.IsSet() {
			v, versionErr := versionutil.StringToVersion(update.Name)
			if versionErr == nil {
				update.Version = v
			}
		}

		// Try getting the application name again because the details
		// line appears to offer a better name sometimes.
		appName := o.GetApplicationName(nextLine)
		if len(appName) > 0 {
			update.ApplicationName = appName
		}

		if strings.Contains(nextLine, restartRequired) {
			update.IsRestartNeeded = true
		}
	}

	return true, update
}

func (o *defaultCliApi) GetApplicationName(line string) string {
	var chars []string

	line = strings.TrimPrefix(line, updateDetailsPrefix)
	line = strings.TrimSpace(line)

	for _, c := range line {
		if unicode.IsNumber(rune(c)) || '(' == c {
			break
		} else {
			chars = append(chars, string(c))
		}
	}

	if len(chars) == 0 {
		return "Unknown Application (" + line + ")"
	}

	return strings.TrimSpace(strings.Join(chars, ""))
}

func (o *defaultCliApi) GetUpdateSizeMegabytes(line string) (uint64, error) {
	parts := strings.Split(line, " ")

	for _, p := range parts {
		if len(p) == 0 {
			continue
		}

		if unicode.IsNumber(rune(p[0])) && strings.HasSuffix(p, updateSizeSuffix) {
			var ints []string

			for _, c := range p {
				if unicode.IsNumber(rune(c)) {
					ints = append(ints, string(c))
				}
			}

			kb, err := strconv.Atoi(strings.Join(ints, ""))
			if err != nil {
				return 0, err
			}

			mb := uint64(kb / 1000)

			if mb == 0 {
				return 1, nil
			}

			return mb, nil
		}
	}

	return 0, errors.New("Could not locate update size")
}

func (o *defaultCliApi) IsInstallProgress(line string) (bool, int) {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, progressPrefix) {
		line = strings.TrimPrefix(line, progressPrefix)
		line = strings.TrimSuffix(line, progressSuffix)

		percent, err := strconv.Atoi(line)
		if err != nil {
			return false, 0
		}

		return true, percent
	}

	return false, 0
}

func (o *defaultCliApi) IsNoSuchUpdate(updateName string, line string) bool {
	if strings.Contains(line, updateName) && strings.HasSuffix(line, noSuchUpdateSuffix) {
		return true
	}

	return false
}

func GetDefaultCliApi() CliApi {
	return &defaultCliApi{
		ExecutablePath: DefaultExecutablePath,
	}
}