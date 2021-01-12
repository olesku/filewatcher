/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

func printUsage(msg string) {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "\tSynchronize path to remote target:\n")
	fmt.Fprintf(os.Stderr, "\t\t%s sync <path-to-sync> <remote-host> <port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tReceive data (listen-mode):\n")
	fmt.Fprintf(os.Stderr, "\t\t%s receive <target-path> <listen-port>\n\n", os.Args[0])

	if len(msg) > 0 {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", msg)
	}

	os.Exit(1)
}

func isValidPort(portStr string) bool {
	n, err := strconv.Atoi(portStr)
	if err != nil || (n < 1 || n > 65535) {
		return false
	}

	return true
}

// Initial file synchronization called before we start
// our fsnotify watcher.
func initialSync(tq *TransferManager, path string) {
	directories := ListDirectories(path)
	files := ListFiles(path)

	for _, dir := range directories {
		tq.Add(QueueItem{
			Action: TmActionMkdir,
			Path:   dir,
		})
	}

	for _, file := range files {
		tq.Add(QueueItem{
			Action: TmActionWrite,
			Path:   file,
		})
	}
}

// sync command entrypoint.
func syncCmd(path string) {
	remote := "127.0.0.1:9090"

	if len(os.Args) == 4 {
		remote = fmt.Sprintf("%s:9090", os.Args[3])
	} else if len(os.Args) == 5 {
		if !isValidPort(os.Args[4]) {
			printUsage(fmt.Sprintf("%s is not a valid port number.", os.Args[4]))
		}
		remote = fmt.Sprintf("%s:%s", os.Args[3], os.Args[4])
	}

	log.Printf("Connecting to %s\n", remote)

	sender := NewSender()
	err := sender.Connect(remote)
	ExitIfError(err)

	txManager := NewTransferManager(sender)

	fileWatcher := NewFileWatcher(txManager)
	err = fileWatcher.Start()
	ExitIfError(err)

	initialSync(txManager, path)
	txManager.Start()
}

// receive command entrypoint.
func receiveCmd(path string) {
	listenAddr := ":9090"

	if len(os.Args) == 4 {
		if !isValidPort(os.Args[3]) {
			printUsage(fmt.Sprintf("%s is not a valid port number.", os.Args[4]))
		}

		listenAddr = fmt.Sprintf(":%s", os.Args[3])
	}

	receiver := NewReceiver()

	log.Printf("Listening on %s\n", listenAddr)
	err := receiver.Start(listenAddr)
	ExitIfError(err)
}

func main() {
	if len(os.Args) < 3 {
		printUsage("Not enough arguments.")
	}

	mode := os.Args[1]
	path := os.Args[2]

	// Check if a valid mode is specified.
	if mode != "sync" && mode != "receive" {
		printUsage(fmt.Sprintf("%s is invalid mode. Supported modes are sync and receive.", mode))
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

	err = os.Chdir(path)
	ExitIfError(err)

	switch mode {
	case "sync":
		syncCmd(path)

	case "receive":
		receiveCmd(path)
	}
}
