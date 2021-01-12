/*
Written by Ole Fredrik Skudsvik <ole.skudsvik@gmail.com> 2021
*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
)

// BlockMeta holds checksum, size and offsets
// for a block in a file.
type BlockMeta struct {
	// Block id (number in chain).
	Index int64

	// Start offset for block.
	Offset int64

	// Block checksum.
	ChkSum string

	// Size of block.
	Size int64
}

// FileMeta holds metadata about a file.
type FileMeta struct {
	Path      string
	Mode      uint32
	Handle    *os.File
	Size      int64
	BlockSize int64
	NumBlocks int64
	Blocks    []BlockMeta
	CheckSum  string
}

// ReadFile reads a file and returns a File object.
func ReadFile(filePath string, blockSize int64) (*FileMeta, error) {
	var f FileMeta

	// If blockSize is set to 0 then set it to 10% of the file size,
	// or to 1MiB if 10% is larger than that.
	// max packet size for gRPC is 4MiB.
	if blockSize == 0 {
		fInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		f.Size = fInfo.Size()
		f.Mode = uint32(fInfo.Mode().Perm())
		blockSize = int64(math.Ceil(float64(fInfo.Size()) / 100 * 10))
		if blockSize > 1024000 {
			blockSize = 1024000
		}
	}

	if err := f.ReadMeta(filePath, blockSize); err != nil {
		return nil, err
	}

	return &f, nil
}

// GetBlockData Get data for given block.
func (f *FileMeta) GetBlockData(blockNumber int64) ([]byte, error) {
	if f.Handle == nil {
		return nil, fmt.Errorf("Read has not been called yet")
	}

	if blockNumber > f.NumBlocks {
		return nil, fmt.Errorf("blockNumber out of range")
	}

	blockSize := int(f.Blocks[int(blockNumber)].Size)
	buf := make([]byte, blockSize)
	nRead, err := f.Handle.ReadAt(buf, f.Blocks[blockNumber].Offset)

	if nRead != blockSize {
		return nil, fmt.Errorf("ReadAt() returned %d bytes, expected %d", nRead, blockSize)
	}

	if err != nil {
		return nil, err
	}

	return buf, nil
}

// GetBlockMeta Get metadata for given block.
func (f *FileMeta) GetBlockMeta(blockNumber int64) (*BlockMeta, error) {
	if f.Handle == nil {
		return nil, fmt.Errorf("Read has not been called yet")
	}

	if blockNumber > f.NumBlocks {
		return nil, fmt.Errorf("blockNumber out of range")
	}

	return &f.Blocks[int(blockNumber)], nil
}

// ReadMeta Read a file and populate a File with metadata about the file and
// its content.
func (f *FileMeta) ReadMeta(filePath string, blockSize int64) (err error) {
	f.Path = filePath

	// Get filesize.
	fInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// If filesize is smaller than our blockSize we reduce blockSize to the
	// size of the file.
	if fInfo.Size() <= blockSize {
		f.NumBlocks = 1
		blockSize = fInfo.Size()
	} else {
		// Round up number of blocks up to nearest integer.
		f.NumBlocks = int64(math.Ceil(float64(fInfo.Size()) / float64(blockSize)))
	}

	// Open file for reading.
	fh, err := os.Open(filePath)
	if err != nil {
		return err
	}

	f.Handle = fh
	f.CheckSum, _ = GetChecksum(filePath)
	f.BlockSize = blockSize

	// Read chunks of blockSize until EOF.
	for i := int64(0); i <= f.NumBlocks; i++ {
		chunk := make([]byte, blockSize)

		var offset int64 = 0

		if i > 0 {
			offset = int64(i * blockSize)
		}

		nRead, err := fh.ReadAt(chunk, offset)

		if err != nil && err != io.EOF {
			return fmt.Errorf("Error reading block %d: %s", i, err.Error())
		}

		if nRead == 0 {
			break
		}

		// Calculate MD5Sum of current block.
		md5Sum := md5.Sum(chunk)

		// Append Block (info) to the block array of our File object.
		f.Blocks = append(f.Blocks, BlockMeta{
			Size:   int64(nRead),
			ChkSum: hex.EncodeToString(md5Sum[:16]),
			Index:  i,
			Offset: offset,
		})
	}

	return nil
}

// Close file handle.
func (f *FileMeta) Close() {
	if f.Handle != nil {
		f.Handle.Close()
	}
}

// GetMissingBlocks returns indexes of missing blocks in file2 compared to file1.
func GetMissingBlocks(file1 *FileMeta, file2 *FileMeta) []int64 {
	var missingBlocks []int64

	// File2 (remote) is empty or larger, return all blocks.
	if file2.Size == 0 || file2.Size > file1.Size {
		for i := int64(0); i < file1.NumBlocks; i++ {
			missingBlocks = append(missingBlocks, i)
		}

		return missingBlocks
	}

	// Files are the same, return 0 blocks.
	if file2.CheckSum == file1.CheckSum {
		return []int64{}
	}

	// Check if any of the blocks at corresponding indexes has mismatching checksums.
	// This diff will only be effective if the number of bytes at any given offset has not changed.
	// Reason being that if bytes are added at one offset every following
	// offset will be shifted. However in many cases this will save data.
	var i int64
	for i = 0; i < file2.NumBlocks && i < file1.NumBlocks; i++ {
		if file2.Blocks[i].ChkSum != file1.Blocks[i].ChkSum {
			missingBlocks = append(missingBlocks, file1.Blocks[i].Index)
		}
	}

	// Append any new blocks at the tail.
	if file1.NumBlocks > file2.NumBlocks {
		for ; i < file1.NumBlocks; i++ {
			missingBlocks = append(missingBlocks, file1.Blocks[i].Index)
		}
	}

	return missingBlocks
}
