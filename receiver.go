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
	recvPath string
}

// NewReceiver Create a new Receiver instance.
func NewReceiver(path string) *Receiver {
	return &Receiver{
		recvPath: path,
	}
}

// Start receiver.
func (r *Receiver) Start(address string) error {
	err := os.Chdir(r.recvPath)
	ExitIfError(err)

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

// CheckFileExist (RPC) Check if a file exists.
func (r *Receiver) CheckFileExist(ctx context.Context, req *FileExistRequest) (*FileExistResponse, error) {
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		return &FileExistResponse{
			Yes: false,
		}, nil
	}

	return &FileExistResponse{
		Yes: true,
	}, nil
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

// DeleteFile (RPC) Delete a file.
func (r *Receiver) DeleteFile(ctx context.Context, req *FileRequest) (*FileRequest, error) {
	err := os.Remove(req.Path)
	return req, err
}

// WriteFileBlock (RPC) Write a chunk of data to a file.
func (r *Receiver) WriteFileBlock(ctx context.Context, req *WriteFileBlockRequest) (*FileExistResponse, error) {
	// TODO: We should cache the filedescriptor and don't reopen it between each call.
	// We should also set the permission to the actual file permission on the sender end.
	fh, err := os.OpenFile(req.GetFilePath(), os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %s\n", req.GetFilePath(), err.Error())
		return &FileExistResponse{}, err
	}

	defer fh.Close()

	_, err = fh.WriteAt(req.GetData(), req.GetOffset())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %d bytes to %s @ offset %d: %s\n", req.GetSize(), req.GetFilePath(), req.GetOffset(), err.Error())
		return &FileExistResponse{}, err
	}

	return &FileExistResponse{}, nil
}

// TruncateFile (RPC) Truncate file at given size.
func (r *Receiver) TruncateFile(ctx context.Context, req *TruncateFileRequest) (*TruncateFileRequest, error) {
	os.Truncate(req.GetPath(), req.GetSize())
	return &TruncateFileRequest{}, nil
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
func (r *Receiver) CreateDirectory(ctx context.Context, req *FileRequest) (*FileRequest, error) {
	err := os.MkdirAll(req.GetPath(), 0777)
	return req, err
}

// DeleteDirectory (RPC) Delete a directory.
func (r *Receiver) DeleteDirectory(ctx context.Context, req *FileRequest) (*FileRequest, error) {
	pInfo, err := os.Stat(req.Path)

	if err != nil {
		return nil, err
	}

	if pInfo.IsDir() {
		err = os.Remove(req.GetPath())
	}

	return req, err
}
