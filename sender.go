/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

// Sender implementation of fileserver.
type Sender struct {
	listener    *net.Listener
	client      ReceiverServiceClient
	isConnected bool
}

// NewSender Create new instance of Sender.
func NewSender() *Sender {
	return &Sender{}
}

// Connect to remote.
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

// Sync Send a file to the remote.
func (s *Sender) Sync(filePath string) error {
	// First compare checksums and exit early if the files are the same.
	localSum, err := GetChecksum(filePath)

	if err != nil {
		return fmt.Errorf("Failed to get checksum for '%s': %s", filePath, err.Error())
	}

	remoteSum, err := s.client.GetFileChecksum(context.Background(), &FileRequest{
		Path: filePath,
	})

	if err == nil && localSum == remoteSum.GetChecksum() {
		return nil
	}

	// Get metadata for the file on the sender end.
	localFile, err := ReadFile(filePath, 0)

	if err != nil {
		return fmt.Errorf("Failed to read '%s': %s", filePath, err.Error())
	}

	// Touch file if it doesn't exist.
	s.Touch(filePath)

	defer localFile.Close()

	// File has no content yet, so we only create it.
	if localFile.Size == 0 {
		s.Chmod(filePath, localFile.Mode)
		return nil
	}

	// Get metadata for the file on the receiver end.
	remoteFile, err := GetRemoteFileMeta(s.client, filePath, localFile.BlockSize)

	// Find the delta between the origin file and the remote.
	var missingBlocks []int64
	if err != nil {
		missingBlocks = GetMissingBlocks(localFile, nil)
	} else {
		// Compare the file on the remote and return the missing blocks.
		missingBlocks = GetMissingBlocks(localFile, remoteFile)
	}

	// Write blocks returned above to the remote.
	for _, blockNum := range missingBlocks {
		blockMeta, err := localFile.GetBlockMeta(blockNum)
		if err != nil {
			return fmt.Errorf("Failed to get meta for block #%d in file '%s': %s", blockNum, filePath, err.Error())
		}

		blockData, err := localFile.GetBlockData(blockNum)
		if err != nil {
			return fmt.Errorf("Failed to get block data for block #%d in file '%s': %s", blockNum, filePath, err.Error())
		}

		_, err = s.client.WriteFileBlock(context.Background(), &WriteFileBlockRequest{
			FilePath: filePath,
			Offset:   blockMeta.Offset,
			Size:     blockMeta.Size,
			Data:     blockData,
		})

		if err != nil {
			return fmt.Errorf("Failed to write to '%s': %s", filePath, err.Error())
		}
	}

	// Truncate file to the correct size.
	_, err = s.client.TruncateFile(context.Background(), &TruncateFileRequest{
		Path: filePath,
		Size: localFile.Size,
	})

	if err != nil {
		return fmt.Errorf("Failed to truncate file '%s' at %d bytes: %s", filePath, localFile.Size, err.Error())
	}

	// Set correct permissions.
	s.Chmod(filePath, localFile.Mode)

	return nil
}

// Touch file if it doesn't exist.
func (s *Sender) Touch(path string) error {
	_, err := s.client.Touch(context.Background(), &FileRequest{
		Path: path,
	})

	return err
}

// Chmod Chmod a file or directory.
func (s *Sender) Chmod(path string, mode uint32) error {
	_, err := s.client.Chmod(context.Background(), &FileRequest{
		Path: path,
		Mode: mode,
	})

	if err != nil {
		return err
	}

	return nil
}

// CreateDirectory Create a directory on the remote.
func (s *Sender) CreateDirectory(path string, mode uint32) error {
	if path == "." || path == ".." {
		return nil
	}

	_, err := s.client.CreateDirectory(context.Background(), &FileRequest{
		Path: path,
		Mode: mode,
	})

	if err != nil {
		return err
	}

	return nil
}

// Delete file or directory.
func (s *Sender) Delete(path string) error {
	_, err := s.client.Delete(context.Background(), &FileRequest{Path: path})

	if err != nil {
		return err
	}

	return nil
}

// Rename file or directory.
func (s *Sender) Rename(oldPath string, newPath string) error {
	_, err := s.client.Rename(context.Background(), &RenameRequest{
		OldPath: oldPath,
		NewPath: newPath,
	})

	return err
}
