package utils

import (
  "os"
  "strings"
  "path/filepath"
  "io/ioutil"
  "os/exec"

)

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

// ForEachGoFilesInDir func
func ForEachGoFilesInDir(path string, fileFunc func(string, error)) {
	filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
    if isGoFile(f) {
  		fileFunc(path, err)
  	}
  	return nil
  })
}

// Diff func
func Diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
