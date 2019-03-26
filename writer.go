package main

import (
	"bytes"

	natomic "github.com/natefinch/atomic"
)

type atomicFileWriter struct {
	hostsFile string
}

func (afw atomicFileWriter) Write(p []byte) (n int, err error) {
	r := bytes.NewReader(p)
	if err := natomic.WriteFile(afw.hostsFile, r); err != nil {
		return 0, err
	}
	return len(p), nil
}
