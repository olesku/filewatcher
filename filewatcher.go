package main

// FileWatcher monitors a directory for changes using inotify.
type FileWatcher struct {
}

// NewFileWatcher Create new instance of FileWatcher.
func (fw *FileWatcher) NewFileWatcher() *FileWatcher {
	return &FileWatcher{}
}
