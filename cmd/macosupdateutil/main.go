package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/stephen-fox/macosupdateutil"
)

const (
	getUpdatesArg    = "g"
	installUpdateArg = "i"
	verboseArg       = "-verbose"
	helpArg          = "h"
)

var (
	getUpdates    = flag.Bool(getUpdatesArg, false, "Get available updates")
	installUpdate = flag.String(installUpdateArg, "", "Install an update by name")
	verbose       = flag.Bool(verboseArg, false, "Run verbose")
	getHelp       = flag.Bool(helpArg, false, "Print this help page")
)

func main() {
	flag.Parse()

	if *getHelp || len(os.Args) == 1 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *getUpdates {
		updates, err := macosupdateutil.GetUpdates()
		if err != nil {
			log.Fatal(err.Error())
		}

		for _, u := range updates {
			fmt.Println("Name: '" + u.Name + "'")

			fmt.Println("Application: '" + u.ApplicationName + "'")

			if u.Version.IsSet() {
				fmt.Println("Version:", u.Version.Long())
			} else {
				fmt.Println("No version avaialble")
			}

			if u.HasUpdateSize() {
				fmt.Println("Update size in mb:", u.SizeMegabytes)
			} else {
				fmt.Println("Update size not available")
			}

			fmt.Println("Is restart needed:", u.IsRestartNeeded)
		}
	}

	if len(strings.TrimSpace(*installUpdate)) > 0 {
		if *verbose {
			progress := make(chan int)

			go func() {
				for percent := range progress {
					fmt.Println(strconv.Itoa(percent) + "%")
				}
			}()

			err := macosupdateutil.InstallUpdateVerbose(*installUpdate, progress)
			if err != nil {
				log.Fatal(err.Error())
			}
		} else {
			err := macosupdateutil.InstallUpdate(*installUpdate)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		log.Println("Finished installing", *installUpdate)
	}
}
