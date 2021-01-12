/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	// TmActionTouch Touch a file.
	TmActionTouch = iota

	// TmActionChmod Chmod a file or directory.
	TmActionChmod = iota

	// TmActionWrite Write to a file.
	TmActionWrite = iota

	// TmActionDelete Delete a file
	TmActionDelete = iota

	// TmActionMkdir Create a directory.
	TmActionMkdir = iota

	// TmActionRename Rename a file or directory.
	TmActionRename = iota
)

// QueueItem Represents a file or directory to be transferred.
type QueueItem struct {
	Path       string
	RenamePath string
	Mode       uint32
	Action     uint
}

// TransferManager Handles queue of files to transfer.
type TransferManager struct {
	sender *Sender
	queue  []QueueItem
	mtx    sync.Mutex
}

// NewTransferManager Create new instance of TransferManager.
func NewTransferManager(sender *Sender) *TransferManager {
	return &TransferManager{
		sender: sender,
	}
}

// Start Transfer queue processor.
func (tq *TransferManager) Start() {
	for {
		tq.processQueue()
		// TODO: Use signaling instead of sleep.
		time.Sleep(100 * time.Millisecond)
	}
}

// Add transfer task.
func (tq *TransferManager) Add(item QueueItem) error {
	filePath, err := StripBasepath(item.Path)

	if err != nil {
		return fmt.Errorf("stripBasePath error: %s", err.Error())
	}

	item.Path = filePath

	tq.mtx.Lock()
	tq.queue = append(tq.queue, item)
	tq.mtx.Unlock()

	return nil
}

// Pop first item off the queue.
func (tq *TransferManager) pop() *QueueItem {
	if len(tq.queue) == 0 {
		return nil
	}

	tq.mtx.Lock()

	it := tq.queue[0]

	if len(tq.queue) > 1 {
		tq.queue = tq.queue[1:]
	} else {
		tq.queue = nil
	}

	tq.mtx.Unlock()
	return &it
}

// Process pendining transfers.
func (tq *TransferManager) processQueue() {
	if !tq.sender.isConnected {
		time.Sleep(1 * time.Second)
		tq.processQueue()
		return
	}

	for {
		item := tq.pop()
		if item == nil {
			return
		}

		switch item.Action {
		case TmActionTouch:
			tq.sender.Touch(item.Path)

		case TmActionChmod:
			tq.sender.Chmod(item.Path, item.Mode)

		case TmActionWrite:
			tq.sender.Sync(item.Path)

		case TmActionMkdir:
			tq.sender.CreateDirectory(item.Path, item.Mode)

		case TmActionDelete:
			tq.sender.Delete(item.Path)

		case TmActionRename:
			newPath, err := StripBasepath(item.RenamePath)

			if err == nil {
				tq.sender.Rename(item.Path, newPath)
			}
		}
	}
}
