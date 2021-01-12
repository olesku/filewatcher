/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors a directory for changes using inotify.
type FileWatcher struct {
	watcher       *fsnotify.Watcher
	tm            *TransferManager
	previousEvent fsnotify.Event
}

// NewFileWatcher Create new instance of FileWatcher.
func NewFileWatcher(transferManager *TransferManager) *FileWatcher {
	return &FileWatcher{
		tm: transferManager,
	}
}

// handleWrite Handler for write event.
func (fw *FileWatcher) handleWrite(event *fsnotify.Event) {
	fw.tm.Add(QueueItem{
		Action: TmActionWrite,
		Path:   event.Name,
	})
}

// handleCreate Handler for create event.
func (fw *FileWatcher) handleCreate(event *fsnotify.Event) {
	if IsDirectory(event.Name) {
		fw.tm.Add(QueueItem{
			Action: TmActionMkdir,
			Path:   event.Name,
			Mode:   GetFileMode(event.Name),
		})
		fw.watcher.Add(event.Name)
	} else {

		// RENAME is sent as two events, first RENAME then CREATE
		// we handle this here.
		if fw.previousEvent.Op == fsnotify.Rename {
			fw.tm.Add(QueueItem{
				Action:     TmActionRename,
				Path:       fw.previousEvent.Name,
				RenamePath: event.Name,
			})
			return
		}

		// File created.
		fw.tm.Add(QueueItem{
			Action: TmActionTouch,
			Path:   event.Name,
		})
	}
}

// handleChmod Handler for chmod event.
func (fw *FileWatcher) handleChmod(event *fsnotify.Event) {
	fInfo, err := os.Stat(event.Name)
	if err != nil {
		return
	}

	fw.tm.Add(QueueItem{
		Action: TmActionChmod,
		Path:   event.Name,
		Mode:   uint32(fInfo.Mode()),
	})
}

// handleRemove Handler for remove event.
func (fw *FileWatcher) handleRemove(event *fsnotify.Event) {
	fw.tm.Add(QueueItem{
		Action: TmActionDelete,
		Path:   event.Name,
	})
}

// Start watching files.
func (fw *FileWatcher) Start() error {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	fw.watcher = watcher

	basePath, err := os.Getwd()
	if err != nil {
		ExitIfError(err)
	}

	watcher.Add(basePath)
	for _, dir := range ListDirectories(basePath) {
		watcher.Add(dir)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				switch event.Op {
				case fsnotify.Write:
					fw.handleWrite(&event)
				case fsnotify.Create:
					fw.handleCreate(&event)
				case fsnotify.Remove:
					fw.handleRemove(&event)
				case fsnotify.Chmod:
					fw.handleChmod(&event)
				}

				fw.previousEvent = event

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Printf("Error fsnotify: %s\n", err)
			}
		}
	}()

	return nil
}
