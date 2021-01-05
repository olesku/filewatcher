package file

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
)

// File contains all blocks of a given file.
type File struct {
	name      string
	blockSize int64
	numBlocks int64
	blocks    []Block
	checkSum  string
}

const (
	// DeltaRemoteLarger f2 is larger.
	DeltaRemoteLarger = iota

	// DeltaEQUAL files are the same.
	DeltaEQUAL = iota

	// DeltaDIFF files differ.
	DeltaDIFF = iota
)

// Delta contains the block differences between two files.
type Delta struct {
	Status int
	Blocks []Block
}

// GetBlocks Return the blocks of the File.
func (f *File) GetBlocks() []Block {
	return f.blocks
}

// GetCheckSum Return the checksum of the File.
func (f *File) GetCheckSum() string {
	return f.checkSum
}

// Read a file and populate File object with (fileSize/blockSize) Blocks containing md5
// checksums and their positional index.
func (f *File) Read(filePath string, blockSize int64) (numblocks int64, err error) {
	f.name = filePath

	// Obtain filesize.
	fInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	// If filesize is smaller than our blockSize we reduce blockSize to the
	// size of the file.
	if fInfo.Size() <= blockSize {
		f.numBlocks = 1
		blockSize = fInfo.Size()
	} else {
		f.numBlocks = int64(math.Ceil(float64(fInfo.Size() / blockSize)))
	}

	// Open file for reading.
	fh, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	defer fh.Close()

	// Get md5checksum of whole file.
	hash := md5.New()
	if _, err := io.Copy(hash, fh); err == nil {
		hashInBytes := hash.Sum(nil)[:16]
		f.checkSum = hex.EncodeToString(hashInBytes)
	}

	// Read chunks of blockSize until EOF.
	for i := int64(0); i <= f.numBlocks; i++ {
		chunk := make([]byte, blockSize)

		var offset int64 = 0

		if i > 0 {
			offset = int64((i * blockSize))
		}

		nRead, err := fh.ReadAt(chunk, offset)

		if err != nil && err != io.EOF {
			fmt.Printf("Error reading block %d: %s\n", i, err.Error())
			return 0, err
		}

		if nRead == 0 {
			break
		}

		// Calculate MD5Sum of current block.
		md5Sum := md5.Sum(chunk)

		// Append Block (info) to the block array of our File object.
		f.blocks = append(f.blocks, Block{
			Size:   int64(nRead),
			ChkSum: md5Sum,
			Index:  i,
		})
	}

	f.blockSize = blockSize
	return int64(len(f.blocks)), nil
}

// GetDelta Compare this File object with another file object
// and return Delta object.
func (f *File) GetDelta(f2 *File) (*Delta, error) {
	// Read has not been called yet, blockSize is not initialized.
	if f.blockSize < 0 {
		return nil, fmt.Errorf("file not initialized with call to Read()")
	}

	// Files have the same checksum, no need to check each block
	// as they are the same.
	if f.checkSum == f2.checkSum {
		return &Delta{
			Status: DeltaEQUAL,
		}, nil
	}

	// If the remote (f2), aka. receiving ends file is bigger than the sender
	// aka. the source (f) then something is not right and we should trigger
	// a full resend.
	if f2.numBlocks > f.numBlocks {
		return &Delta{
			Status: DeltaRemoteLarger,
		}, nil
	}

	var delta Delta
	delta.Status = DeltaDIFF

	// First check if any of the blocks at the same indexes has changed it's content.
	// This diff will only be effective if the number of bytes has not changed.
	// Reason being that if bytes are added at one offset every following
	// offset will be shifted. However in some cases this might save data.
	var i int64
	for i = 0; i < int64(len(f2.blocks)); i++ {
		if f2.blocks[i].ChkSum != f.blocks[i].ChkSum {
			delta.Blocks = append(delta.Blocks, f.blocks[i])
		}
	}

	// Append any new blocks at the tail.
	if f.numBlocks > f2.numBlocks {
		for ; i < f.numBlocks; i++ {
			delta.Blocks = append(delta.Blocks, f.blocks[i])
		}
	}

	return &delta, nil
}
