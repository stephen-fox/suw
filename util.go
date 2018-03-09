package macosupdateutil

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/stephen-fox/versionutil"
)

const (
	executableParentPath  = "/usr/sbin/"
	executableName        = "softwareupdate"
	defaultExecutablePath = executableParentPath + executableName

	updatePrefix        = "*"
	updateDetailsPrefix = "\t"
	updateSizeSuffix    = "K"
	restartRequired     = "[restart]"

	listArg    = "-l"
	installArg = "-i"
	verboseArg = "--verbose"
)

/*
$ softwareupdate -l
Software Update Tool

Finding available software
Software Update found the following new or updated software:
   * iTunesX-12.7.3
	iTunes (12.7.3), 264516K [recommended]

####

$ softwareupdate -i 'iTunesX-12.7.3'
Software Update Tool


Downloaded iTunes
Installing iTunes
Done with iTunes
Done.
 */

type Update struct {
	ReferenceName   string
	ApplicationName string
	Version         versionutil.Version
	SizeMegabytes   uint64
	IsRestartNeeded bool
}

func (o Update) HasUpdateSize() bool {
	return o.SizeMegabytes > 0
}

type runResult struct {
	Command  string
	ExitCode int
	Output   []string
	Error    string
}

func (o runResult) isErr() bool {
	return o.ExitCode > 0
}

func (o runResult) getErr() error {
	message := "Failed to execute '" + o.Command + "' - " + o.Error

	if len(o.Output) > 0 {
		message = message + ". Output: " + strings.Join(o.Output, " ")
	}

	return errors.New(message)
}

func GetUpdates() ([]Update, error) {
	rr := run(listArg)
	if rr.isErr() {
		return []Update{}, rr.getErr()
	}

	var updates []Update

	for i, l := range rr.Output {
		// We cannot remove trailing spaces because softwareupdate
		// treats them as part of the update name.
		noLeadingSpaces := strings.TrimLeft(l, " ")

		if strings.HasPrefix(noLeadingSpaces, updatePrefix) {
			refName := strings.TrimPrefix(noLeadingSpaces, updatePrefix + " ")
			appName := getApplicationName(refName)

			update := Update{
				ReferenceName:   refName,
				ApplicationName: appName,
			}

			v, versionErr := versionutil.StringToVersion(update.ReferenceName)
			if versionErr == nil {
				update.Version = v
			}

			if i < len(rr.Output) - 1 {
				nextLine := rr.Output[i+1]

				if !strings.HasPrefix(nextLine, updateDetailsPrefix) {
					updates = append(updates, update)
					continue
				}

				mb, sizeErr := getUpdateSizeMegabytes(nextLine)
				if sizeErr == nil {
					update.SizeMegabytes = mb
				}

				if !update.Version.IsSet() {
					v, versionErr := versionutil.StringToVersion(update.ReferenceName)
					if versionErr == nil {
						update.Version = v
					}
				}

				// Try getting the application name again because the details
				// line appears to offer a better name sometimes.
				appName := getApplicationName(nextLine)
				if len(appName) > 0 {
					update.ApplicationName = appName
				}

				if strings.Contains(nextLine, restartRequired) {
					update.IsRestartNeeded = true
				}
			}

			updates = append(updates, update)
		}
	}

	return updates, nil
}

func getApplicationName(str string) string {
	var chars []string

	str = strings.TrimPrefix(str, updateDetailsPrefix)
	str = strings.TrimSpace(str)

	for _, c := range str {
		if unicode.IsNumber(rune(c)) || '(' == c {
			break
		} else {
			chars = append(chars, string(c))
		}
	}

	if len(chars) == 0 {
		return "Unknown Application (" + str + ")"
	}

	return strings.TrimSpace(strings.Join(chars, ""))
}

func getUpdateSizeMegabytes(str string) (uint64, error) {
	parts := strings.Split(str, " ")

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

			return uint64(kb / 1000), nil
		}
	}

	return 0, errors.New("Could not locate update size")
}

func run(args ...string) runResult {
	command := exec.Command(defaultExecutablePath, args...)

	raw, err := command.CombinedOutput()
	fullOutput := string(raw)
	var output []string
	if len(strings.TrimSpace(fullOutput)) > 0 {
		output = strings.Split(fullOutput, "\n")
	}

	rr := runResult{
		Command: command.Path + " " + strings.Join(command.Args, " "),
		Output:  output,
	}

	if err != nil {
		rr.Error = err.Error()
		rr.ExitCode = 1
	}

	return rr
}