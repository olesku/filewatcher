/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

// Sender implementation of fileserver.
type Sender struct {
	listener *net.Listener
	syncPath string
	client   ReceiverServiceClient
}

// NewSender Create a new Sender instance.
func NewSender(path string) *Sender {
	return &Sender{
		syncPath: path,
	}
}

// Start - Connect to remote.
func (s *Sender) Start(address string) error {
	err := os.Chdir(s.syncPath)
	ExitIfError(err)

	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", address, err.Error())
	}

	s.client = NewReceiverServiceClient(conn)

	// Initial sync.
	s.SyncDirectory(".")

	return nil
}

// SyncFile Send a file to the remote.
func (s *Sender) SyncFile(filePath string) {
	// Get metadata for the file on the sender end.
	localFile, err := ReadFile(filePath, 0)

	if err != nil {
		log.Printf("Failed to read %s: %s\n", filePath, err.Error())
		return
	}

	defer localFile.Close()

	// Get metadata for the file on the receiver end.
	var missingBlocks []int64
	remoteFile, err := GetRemoteFileMeta(s.client, filePath, localFile.BlockSize)
	if err != nil {
		for i := int64(0); i < localFile.NumBlocks; i++ {
			missingBlocks = append(missingBlocks, i)
		}
	} else {
		// Compare the file on the remote if it exists and return
		// the missing blocks. If file does not exist on remote
		// all blocks is returned.
		missingBlocks = GetMissingBlocks(localFile, remoteFile)
	}

	if len(missingBlocks) > 0 {
		log.Printf("Transferring %s\n", filePath)
	}

	// Write blocks returned above to the remote.
	for _, blockNum := range missingBlocks {
		blockMeta, err := localFile.GetBlockMeta(blockNum)
		if err != nil {
			log.Printf("Failed to get meta for block #%d in file %s: %s\n", blockNum, filePath, err.Error())
			return
		}

		blockData, err := localFile.GetBlockData(blockNum)
		if err != nil {
			log.Printf("Failed to get block data for block #%d in file %s: %s\n", blockNum, filePath, err.Error())
			return
		}

		//fmt.Printf("Writing block #%d (%d bytes)\n", blockNum, blockMeta.Size)
		_, err = s.client.WriteFileBlock(context.Background(), &WriteFileBlockRequest{
			FilePath: filePath,
			Offset:   blockMeta.Offset,
			Size:     blockMeta.Size,
			Data:     blockData,
		})

		if err != nil {
			log.Printf("Failed to write to %s: %s\n", filePath, err.Error())
			return
		}
	}

	// Truncate file to the correct size.
	_, err = s.client.TruncateFile(context.Background(), &TruncateFileRequest{
		Path: filePath,
		Size: localFile.Size,
	})

	if err != nil {
		log.Printf("Failed to truncate %s at %d bytes: %s\n", filePath, localFile.Size, err.Error())
		return
	}
}

// SyncDirectory Transfer a directory to the remote.
func (s *Sender) SyncDirectory(path string) {
	directories := ListDirectories(path)
	files := ListFiles(path)

	for _, dir := range directories {
		_, err := s.client.CreateDirectory(context.Background(), &FileRequest{Path: dir})
		if err != nil {
			log.Printf("Failed to create directory %s: %s\n", dir, err.Error())
		} else {
			log.Printf("Creating directory %s\n", dir)
		}
	}

	for _, file := range files {
		s.SyncFile(file)
	}
}
