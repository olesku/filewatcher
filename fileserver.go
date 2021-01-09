package main

// FileServer Base interface.
type FileServer interface {
	Start(address string) error
}
