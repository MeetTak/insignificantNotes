package main

import (
	"flag"
	"fmt"
	"os"
)

func newFlags(name, about string) *flag.FlagSet {
	fl := flag.NewFlagSet(name, flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: notesync %s [flags]\n\n%s\n\nFlags:\n", name, about)
		fl.PrintDefaults()
	}
	return fl
}
