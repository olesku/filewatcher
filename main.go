/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"fmt"
	"log"
	"os"
)

func printUsage(msg string) {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "\tSynchronize path to remote target:\n")
	fmt.Fprintf(os.Stderr, "\t\t%s <serve> <path> <remote-host> <port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tReceive data (listen-mode):\n")
	fmt.Fprintf(os.Stderr, "\t\t%s <receive> <path> <listen-port>\n\n", os.Args[0])

	if len(msg) > 0 {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", msg)
	}

	os.Exit(1)
}

func main() {
	var server FileServer
	if len(os.Args) < 3 {
		printUsage("Not enough arguments.")
	}

	mode := os.Args[1]
	path := os.Args[2]

	// Check if a valid mode is specified.
	if mode != "serve" && mode != "receive" {
		printUsage(fmt.Sprintf("%s is invalid mode. Supported modes are serve and receive.", mode))
	}

	fInfo, err := os.Stat(path)

	// Check if specified path is a directory.
	if err != nil {
		if os.IsNotExist(err) {
			printUsage(fmt.Sprintf("%s is not a directory.", path))
		} else {
			printUsage(fmt.Sprintf("%s: %s.", path, err.Error()))
		}
	}

	if !fInfo.IsDir() {
		printUsage(fmt.Sprintf("%s is not a directory.", path))
	}

	// Instantiate correct server implementation for specified mode.
	switch mode {
	case "serve":
		server = NewSender(path)

	case "receive":
		server = NewReceiver(path)
	}

	// Start server.
	if err := server.Start(":9999"); err != nil {
		log.Fatalf("Failed to start: %s\n", err.Error())
	}
}
