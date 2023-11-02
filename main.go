package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	// "github.com/syssecfsu/witty/cmd"
	// "github.com/syssecfsu/witty/web"
	"lannerkr/witty/cmd"
	"lannerkr/witty/web"
)

// https://github.com/lannerkr/web-term
// version 1.1

const (
	// subcmds = "witty (adduser|deluser|listusers|replay|merge|run)"
	subcmds = "witty (adduser|deluser|listusers|run)"
)

//go:embed assets/*
var fullAssets embed.FS

func main() {
	if len(os.Args) < 2 {
		fmt.Println(subcmds)
		return
	}

	switch os.Args[1] {
	case "adduser":
		if len(os.Args) != 3 {
			fmt.Println("witty adduser <username>")
			return
		}
		cmd.AddUser(os.Args[2])

	case "deluser":
		if len(os.Args) != 3 {
			fmt.Println("witty deluser <username>")
			return
		}
		cmd.DelUser(os.Args[2])

	case "listusers":
		cmd.ListUsers()

	case "run":
		// setup the web options
		var options web.Options
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		runCmd.BoolVar(&options.NoAuth, "n", false, "Run WiTTY without user authentication")
		runCmd.BoolVar(&options.NoAuth, "naked", false, "Run WiTTY without user authentication")
		runCmd.UintVar(&options.Port, "p", 8090, "Port number to listen on")
		runCmd.UintVar(&options.Port, "port", 8090, "Port number to listen on")
		runCmd.UintVar(&options.Wait, "w", 1000, "Max wait time between outputs")
		runCmd.UintVar(&options.Wait, "wait", 1000, "Max wait time between outputs")

		fp, err := os.OpenFile("witty.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

		if err == nil {
			defer fp.Close()
			log.SetOutput(fp)
		}

		options.LogFile = fp

		runCmd.Parse(os.Args[2:])

		var cmdToExec []string
		args := runCmd.Args()
		if len(args) > 0 {
			cmdToExec = args
		} else {
			// cmdToExec = []string{"bash"}
			cmdToExec = []string{"sudo", "-H", "-u", "lannerkr", "bash"}
		}

		options.CmdToExec = cmdToExec

		// we need to strip the top level directory for Gin to find the files
		assets, err := fs.Sub(fullAssets, "assets")

		if err != nil {
			log.Fatal("Failed to load assets", err)
		}

		options.Assets = assets

		web.StartWeb(&options)

	default:
		fmt.Println(subcmds)
		return
	}

}
