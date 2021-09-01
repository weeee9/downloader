package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func getDir(filename string) string {
	return strings.SplitN(filename, ".", 2)[0]
}

func getPartialFilename(filename string, partNum int) string {
	partDir := getDir(filename)
	filename = fmt.Sprintf("%s-%d", filename, partNum)
	return filepath.Join(partDir, filename)
}
