/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"fmt"
	"os"
)

func printUsage(msg string) {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "\tSynchronize path to remote target:\n")
	fmt.Fprintf(os.Stderr, "\t\t%s <send> <path> <remote-host> <port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tReceive data (listen-mode):\n")
	fmt.Fprintf(os.Stderr, "\t\t%s <receive> <path> <listen-port>\n\n", os.Args[0])

	if len(msg) > 0 {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", msg)
	}

	os.Exit(1)
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

func main() {
	if len(os.Args) < 3 {
		printUsage("Not enough arguments.")
	}

	mode := os.Args[1]
	path := os.Args[2]

	// Check if a valid mode is specified.
	if mode != "send" && mode != "receive" {
		printUsage(fmt.Sprintf("%s is invalid mode. Supported modes are send and receive.", mode))
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
	case "send":
		sender := NewSender()
		err := sender.Connect(":9999")
		ExitIfError(err)

		txManager := NewTransferManager(sender)

		fileWatcher := NewFileWatcher(txManager)
		err = fileWatcher.Start()
		ExitIfError(err)

		initialSync(txManager, path)
		txManager.Start()

	case "receive":
		receiver := NewReceiver()
		err := receiver.Start(":9999")
		ExitIfError(err)
	}
}
