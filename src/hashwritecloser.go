package	main


import	(
	"io"
	"os"
	"fmt"
	"hash"
	"strings"

	sd "github.com/nathanaelle/sdialog"
)

type	HashedWriteCloser struct {
	name		string
	h		hash.Hash
	pipe		io.WriteCloser
}


func NewHashedWriteCloser(file string, h hash.Hash) (*HashedWriteCloser) {
	pipe, err	:= os.Create(file)
	if err != nil {
		panic(err)
	}
	sd.Status(strings.Join([]string{ file, "opened" }," "))

	return &HashedWriteCloser {
		h:	h,
		name:	file,
		pipe:	pipe,
	}
}


func (hwc *HashedWriteCloser) Close() error {
	err := hwc.pipe.Close()
	sum := hwc.h.Sum(nil)
	sd.Status(strings.Join([]string{ hwc.name, "closed hash [",  fmt.Sprintf("%x", sum) ,"]" }," "))
	return	err
}


func (hwc *HashedWriteCloser) Write(b []byte) (s int, e error) {
	s,e = hwc.pipe.Write(b)
	hwc.h.Write(b[0:s])
	return s,e
}
