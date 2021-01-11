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

func (fw *FileWatcher) handleWrite(event *fsnotify.Event) {
	fw.tm.Add(QueueItem{
		Action: TmActionWrite,
		Path:   event.Name,
	})
}

func (fw *FileWatcher) handleCreate(event *fsnotify.Event) {
	if IsDirectory(event.Name) {
		fw.tm.Add(QueueItem{
			Action: TmActionMkdir,
			Path:   event.Name,
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

func (fw *FileWatcher) handleDelete(event *fsnotify.Event) {
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

				log.Printf("Event: %s\n", event)

				if event.Op == fsnotify.Write {
					fw.handleWrite(&event)
				} else if event.Op == fsnotify.Create {
					fw.handleCreate(&event)
				} else if event.Op == fsnotify.Remove {
					fw.handleDelete(&event)
				} else if event.Op == fsnotify.Chmod {
					fw.handleChmod(&event)
				}

				fw.previousEvent = event

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Printf("Error: %s\n", err)
			}
		}
	}()

	return nil
}
