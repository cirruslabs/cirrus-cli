package unexpectedeofreader

import "io"

type UnexpectedEOFReader struct {
	inner    io.Reader
	numReads int
}

func New(r io.Reader) io.Reader {
	return &UnexpectedEOFReader{
		inner: r,
	}
}

func (f *UnexpectedEOFReader) Read(p []byte) (int, error) {
	f.numReads += 1

	// Return 1, then 2, then 3 bytes of the actual content for the first three reads
	if f.numReads <= 3 {
		return f.inner.Read(p[:f.numReads])
	}

	// Then return unexpected EOF
	return 0, io.ErrUnexpectedEOF
}
