package main

import (
	"log"

	"github.com/olesku/filewatcher/file"
)

func main() {
	var f1 file.File
	var f2 file.File

	n, err := f1.Read("./Makefile2", 4)

	if err != nil {
		log.Fatalf("f1: Error: %s\n", err.Error())
	}

	log.Printf("f1: Read %d blocks\n", n)
	log.Printf("f1: Checksum: %s\n", f1.GetCheckSum())

	n, err = f2.Read("./Makefile", 4)

	if err != nil {
		log.Fatalf("f2: Error: %s\n", err.Error())
	}

	log.Printf("f2: Read %d blocks\n", n)
	log.Printf("f2: Checksum: %s\n", f2.GetCheckSum())

	diff, err := f1.GetDelta(&f2)

	if err == nil {
		log.Printf("Delta:\nStatus: %d\nNumBlocks (diff): %d\nBlocks:\n", diff.Status, len(diff.Blocks))

		for _, b := range diff.Blocks {
			log.Printf("%d\n", b.Index)
		}
	}
}
