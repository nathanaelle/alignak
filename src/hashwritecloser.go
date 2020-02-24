package main

import (
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	sd "github.com/nathanaelle/sdialog/v2"
)

type hashedWriteCloser struct {
	name string
	h    hash.Hash
	pipe io.WriteCloser
}

func newHashedWriteCloser(file string, h hash.Hash) io.WriteCloser {
	pipe, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	sd.Status(strings.Join([]string{file, "opened"}, " "))

	return &hashedWriteCloser{
		h:    h,
		name: file,
		pipe: pipe,
	}
}

func (hwc *hashedWriteCloser) Close() error {
	err := hwc.pipe.Close()
	sum := hwc.h.Sum(nil)
	sd.Status(strings.Join([]string{hwc.name, "closed hash [", fmt.Sprintf("%x", sum), "]"}, " "))
	return err
}

func (hwc *hashedWriteCloser) Write(b []byte) (s int, e error) {
	s, e = hwc.pipe.Write(b)
	hwc.h.Write(b[0:s])
	return s, e
}
