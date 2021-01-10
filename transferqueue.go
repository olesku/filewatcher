package main

import (
	sync "sync"
	"time"
)

// QueueItem Represents a file or directory to be transferred.
type QueueItem struct {
	isDir bool
	path  string
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

// AddFile Add file to queue.
func (tq *TransferManager) AddFile(filePath string) {
	tq.mtx.Lock()
	tq.queue = append(tq.queue, QueueItem{
		path:  filePath,
		isDir: false,
	})

	tq.mtx.Unlock()
}

// CreateDirectory Queue creation of a directory.
func (tq *TransferManager) CreateDirectory(dir string) {
	tq.mtx.Lock()

	tq.queue = append(tq.queue, QueueItem{
		path:  dir,
		isDir: true,
	})

	tq.mtx.Unlock()
}

// Pop first item off the queue.
func (tq *TransferManager) pop() *QueueItem {
	tq.mtx.Lock()

	if len(tq.queue) == 0 {
		return nil
	}

	it := tq.queue[0]

	if len(tq.queue) > 1 {
		tq.queue = tq.queue[1:]
	} else {
		tq.queue = nil
	}

	tq.mtx.Unlock()
	return &it
}

// processQueue Process pendining transfers.
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

		// TODO: Implement retries if send fails.
		if item.isDir {
			tq.sender.CreateDirectory(item.path)
		} else {
			tq.sender.SyncFile(item.path)
		}
	}
}
