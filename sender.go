/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

// Sender implementation of fileserver.
type Sender struct {
	listener    *net.Listener
	client      ReceiverServiceClient
	isConnected bool
}

// NewSender Create news instance of Sender.
func NewSender() *Sender {
	return &Sender{}
}

// Connect - Connect to remote.
func (s *Sender) Connect(address string) error {
	s.isConnected = false
	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		return fmt.Errorf("Failed to connect to %s: %s", address, err.Error())
	}

	s.client = NewReceiverServiceClient(conn)
	s.isConnected = true

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
		// If file was not found on remote end we should just write all blocks.
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

// CreateDirectory ..
func (s *Sender) CreateDirectory(path string) error {
	_, err := s.client.CreateDirectory(context.Background(), &FileRequest{Path: path})

	if err != nil {
		return err
	}

	return nil
}
