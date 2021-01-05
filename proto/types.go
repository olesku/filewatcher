package proto

// FileTransferInfo holds information about a file that is going
// to be transferred to our remote counterpart.
type FileTransferInfo struct {
	blockIndexList []int64
}
