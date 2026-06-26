package finder

import (
	"io/fs"
	"os"
)

// shouldIgnore reports whether the entry should be filtered out per the
// configured file-type constraints.
func (ft *FileTypes) shouldIgnore(e *DirEntry) bool {
	mode := e.Type()
	// If we couldn't determine a type, ignore it.
	info, err := e.Info()
	if err != nil && mode == 0 {
		return true
	}
	if mode == 0 && info != nil {
		mode = info.Mode().Type()
	}

	isDir := mode.IsDir()
	isFile := mode.IsRegular()
	isSymlink := mode&fs.ModeSymlink != 0
	isBlock := mode&fs.ModeDevice != 0 && mode&fs.ModeCharDevice == 0
	isChar := mode&fs.ModeCharDevice != 0
	isSocket := mode&fs.ModeSocket != 0
	isPipe := mode&fs.ModeNamedPipe != 0

	if !ft.Files && isFile {
		return true
	}
	if !ft.Directories && isDir {
		return true
	}
	if !ft.Symlinks && isSymlink {
		return true
	}
	if !ft.BlockDevices && isBlock {
		return true
	}
	if !ft.CharDevices && isChar {
		return true
	}
	if !ft.Sockets && isSocket {
		return true
	}
	if !ft.Pipes && isPipe {
		return true
	}
	if ft.ExecutablesOnly && !isExecutable(e) {
		return true
	}
	if ft.EmptyOnly && !isEmpty(e) {
		return true
	}
	// Unknown type
	if !(isFile || isDir || isSymlink || isBlock || isChar || isSocket || isPipe) {
		return true
	}
	return false
}

func isExecutable(e *DirEntry) bool {
	info, err := e.Info()
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0111 != 0
}

func isEmpty(e *DirEntry) bool {
	info, err := e.Info()
	if err != nil {
		return false
	}
	if info.IsDir() {
		f, err := os.Open(e.path)
		if err != nil {
			return false
		}
		defer f.Close()
		_, err = f.Readdirnames(1)
		return err != nil // io.EOF => empty
	}
	if info.Mode().IsRegular() {
		return info.Size() == 0
	}
	return false
}
