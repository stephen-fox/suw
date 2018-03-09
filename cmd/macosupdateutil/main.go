package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/stephen-fox/macosupdateutil"
)

const (
	getUpdatesArg = "g"
	helpArg       = "h"
)

var (
	getUpdates = flag.Bool(getUpdatesArg, false, "Get available updates")

	getHelp = flag.Bool(helpArg, false, "Print this help page")
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
			fmt.Println("Reference: '" + u.ReferenceName + "'")

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
}
