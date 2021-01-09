package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListDirectories(t *testing.T) {
	t.Logf("Foo\n")

	dirs := ListDirectories("/home/oles/www")

	for _, dir := range dirs {
		t.Logf("Dir: %s\n", dir)
	}

	assert.NotEmpty(t, dirs)
}

func TestListFiles(t *testing.T) {
	t.Logf("Foo\n")

	files := ListFiles(".")

	for _, file := range files {
		t.Logf("File: %s\n", file)
	}

	assert.NotEmpty(t, files)
}
