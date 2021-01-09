package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileOps(t *testing.T) {
	testData := []byte("AAAABBBBCCCCDDDDEEE")

	fh, err := ioutil.TempFile("/tmp", "filewatcher_test")
	if err != nil {
		t.Errorf("Failed to create temporary testdata file: %s\n", err.Error())
	}

	filePath := fh.Name()
	fh.Write(testData)
	fh.Close()

	defer os.Remove(filePath)

	f, err := ReadFile(filePath, 4)
	assert.NoError(t, err, "No error when calling ReadFile")
	assert.Equal(t, int64(5), f.NumBlocks, "Number of blocks should equal to 5")

	b1, err := f.GetBlockMeta(0)
	assert.NoError(t, err, "No error should be returned when calling GetBlockMeta(0)")
	assert.Equal(t, int64(4), b1.Size, "Block #0 should have size of 4")

	b4, err := f.GetBlockMeta(4)
	assert.NoError(t, err, "No error should be returned when calling GetBlockMeta(4)")
	assert.Equal(t, int64(3), b4.Size, "Block #4 should have size of 3")

	data, err := f.GetBlockData(4)
	assert.NoError(t, err, "No error when calling GetBlockData(4)")
	assert.Equal(t, data, []byte("EEE"), data, "Last block should contain 123")

	assert.Equal(t, "7f0a7164fcaaadb4559d0f842bb35dd3", f.CheckSum, "Checksum should be correct")
}
