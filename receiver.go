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

// Receiver implementation of fileserver.
type Receiver struct {
	listener *net.Listener
	grpcSrv  *grpc.Server
}

// NewReceiver Create a new Receiver instance.
func NewReceiver() *Receiver {
	return &Receiver{}
}

// Start receiver.
func (r *Receiver) Start(address string) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return fmt.Errorf("Failed to listen: %s", err.Error())
	}

	grpcSrv := grpc.NewServer()

	if err != nil {
		return fmt.Errorf("Failed to start gRPC server: %s", err.Error())
	}

	r.listener = &listener
	r.grpcSrv = grpcSrv

	RegisterReceiverServiceServer(grpcSrv, &Receiver{})
	err = grpcSrv.Serve(listener)

	log.Printf("Listening on %s\n", address)

	return nil
}

// GetFileChecksum (RPC) Get MD5 checksum of a file.
func (r *Receiver) GetFileChecksum(ctx context.Context, req *FileRequest) (*FileChecksumResponse, error) {
	checkSum, err := GetChecksum(req.GetPath())
	if err != nil {
		return &FileChecksumResponse{}, err
	}

	return &FileChecksumResponse{
		Checksum: checkSum,
	}, nil
}

// Rename (RPC) Rename a file or directory.
func (r *Receiver) Rename(ctx context.Context, req *RenameRequest) (*EmptyResponse, error) {
	err := os.Rename(req.GetOldPath(), req.GetNewPath())

	return &EmptyResponse{}, err
}

// Delete (RPC) Delete a file or directory.
func (r *Receiver) Delete(ctx context.Context, req *FileRequest) (*EmptyResponse, error) {
	err := os.RemoveAll(req.Path)
	return &EmptyResponse{}, err
}

// Touch (RPC) Create a file if it doesn't exist and set correct permissions.
func (r *Receiver) Touch(ctx context.Context, req *FileRequest) (*EmptyResponse, error) {
	_, err := os.Stat(req.GetPath())

	if err != nil && os.IsNotExist(err) {
		fh, err := os.Create(req.GetPath())
		fh.Close()

		if err != nil {
			return &EmptyResponse{}, err
		}
	}

	return &EmptyResponse{}, nil
}

// Chmod (RPC) Chmod a file or directory.
func (r *Receiver) Chmod(ctx context.Context, req *FileRequest) (*EmptyResponse, error) {
	err := os.Chmod(req.GetPath(), os.FileMode(req.GetMode()))
	return &EmptyResponse{}, err
}

// WriteFileBlock (RPC) Write a chunk of data to a file.
func (r *Receiver) WriteFileBlock(ctx context.Context, req *WriteFileBlockRequest) (*EmptyResponse, error) {
	// TODO: We should cache the filedescriptor and don't reopen it between each call.
	fh, err := os.OpenFile(req.GetFilePath(), os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %s\n", req.GetFilePath(), err.Error())
		return &EmptyResponse{}, err
	}

	defer fh.Close()

	_, err = fh.WriteAt(req.GetData(), req.GetOffset())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %d bytes to %s @ offset %d: %s\n", req.GetSize(), req.GetFilePath(), req.GetOffset(), err.Error())
		return &EmptyResponse{}, err
	}

	return &EmptyResponse{}, nil
}

// TruncateFile (RPC) Truncate file at given size.
func (r *Receiver) TruncateFile(ctx context.Context, req *TruncateFileRequest) (*EmptyResponse, error) {
	os.Truncate(req.GetPath(), req.GetSize())
	return &EmptyResponse{}, nil
}

// GetFileMeta (RPC) Get metadata of file.
func (r *Receiver) GetFileMeta(ctx context.Context, req *FileRequest) (*FileResponse, error) {
	f, err := ReadFile(req.GetPath(), req.GetBlockSize())
	if err != nil {
		return &FileResponse{}, err
	}

	defer f.Close()

	var blockMetaList []*BlockMetaType

	for _, block := range f.Blocks {
		blockMetaList = append(blockMetaList, &BlockMetaType{
			Index:  block.Index,
			Offset: block.Offset,
			ChkSum: block.ChkSum,
			Size:   block.Size,
		})
	}

	resp := &FileResponse{
		Path:      f.Path,
		BlockSize: f.BlockSize,
		NumBlocks: f.NumBlocks,
		BlockMeta: blockMetaList,
		CheckSum:  f.CheckSum,
	}

	return resp, nil
}

// GetRemoteFileMeta (RPC) Marshals the response from GetFileMeta into a FileMeta object.
func GetRemoteFileMeta(client ReceiverServiceClient, filePath string, blockSize int64) (*FileMeta, error) {
	resp, err := client.GetFileMeta(context.Background(), &FileRequest{
		Path:      filePath,
		BlockSize: blockSize,
	})

	if err != nil {
		return nil, err
	}

	var blockList []BlockMeta
	for _, blockMeta := range resp.GetBlockMeta() {
		blockList = append(blockList, BlockMeta{
			Index:  blockMeta.GetIndex(),
			Offset: blockMeta.GetOffset(),
			ChkSum: blockMeta.GetChkSum(),
			Size:   blockMeta.GetSize(),
		})
	}

	return &FileMeta{
		Path:      resp.GetPath(),
		Handle:    nil,
		BlockSize: resp.GetBlockSize(),
		NumBlocks: resp.GetNumBlocks(),
		Blocks:    blockList,
		CheckSum:  resp.GetCheckSum(),
	}, nil
}

// CreateDirectory (RPC) Create a directory.
func (r *Receiver) CreateDirectory(ctx context.Context, req *FileRequest) (*EmptyResponse, error) {
	// TODO:  Set correct permission.
	err := os.MkdirAll(req.GetPath(), 0777)
	return &EmptyResponse{}, err
}
