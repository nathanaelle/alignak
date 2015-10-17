package	main


import	(
	"io"
	"os"
	"fmt"
	"hash"
	"strings"

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
	sd_notify("STATUS",strings.Join([]string{ file, "opened" }," "))

	return &HashedWriteCloser {
		h:	h,
		name:	file,
		pipe:	pipe,
	}
}

func (hwc *HashedWriteCloser) Close()  {
	hwc.pipe.Close()
	sum := hwc.h.Sum(nil)
	sd_notify("STATUS",strings.Join([]string{ hwc.name, "closed hash [",  fmt.Sprintf("%x", sum) ,"]" }," "))
}

func (hwc *HashedWriteCloser) Write(b []byte) (s int, e error) {
	s,e = hwc.pipe.Write(b)
	hwc.h.Write(b[0:s])
	return s,e
}
