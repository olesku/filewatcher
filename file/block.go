package file

// Block contains a checksum and positional information
// about a block in a file.
type Block struct {
	Index  int64
	ChkSum [16]byte
	Size   int64
}
