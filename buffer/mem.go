package buffer

import (
	"bytes"
	"io"

	"github.com/djherbis/buffer/limio"
)

type memory struct {
	N int64
	*bytes.Buffer
}

// Buffer is used to Write() data which will be Read() later.
type Buffer interface {
	Len() int64 // How much data is Buffered in bytes
	Cap() int64 // How much data can be Buffered at once in bytes.
	io.Reader   // Read() will read from the top of the buffer [io.EOF if empty]
	io.Writer   // Write() will write to the end of the buffer [io.ErrShortWrite if not enough space]
	Reset()     // Truncates the buffer, Len() == 0.
}

// BufferAt is a buffer which supports io.ReaderAt and io.WriterAt
type BufferAt interface {
	Buffer
	io.ReaderAt
	io.WriterAt
}

func len64(p []byte) int64 {
	return int64(len(p))
}

// Gap returns buf.Cap() - buf.Len()
func Gap(buf Buffer) int64 {
	return buf.Cap() - buf.Len()
}

// New returns a new in memory BufferAt with max size N.
// It's backed by a bytes.Buffer.
func New(n int64) BufferAt {
	return &memory{
		N:      n,
		Buffer: bytes.NewBuffer(nil),
	}
}

func (buf *memory) Cap() int64 {
	return buf.N
}

func (buf *memory) Len() int64 {
	return int64(buf.Buffer.Len())
}

func (buf *memory) Write(p []byte) (n int, err error) {
	return limio.LimitWriter(buf.Buffer, Gap(buf)).Write(p)
}

func (buf *memory) WriteAt(p []byte, off int64) (n int, err error) {
	if off > buf.Len() {
		return 0, io.ErrShortWrite
	} else if len64(p)+off <= buf.Len() {
		d := buf.Bytes()[off:]
		return copy(d, p), nil
	} else {
		d := buf.Bytes()[off:]
		n = copy(d, p)
		m, err := buf.Write(p[n:])
		return n + m, err
	}
}

func (buf *memory) ReadAt(p []byte, off int64) (n int, err error) {
	return bytes.NewReader(buf.Bytes()).ReadAt(p, off)
}

func (buf *memory) Read(p []byte) (n int, err error) {
	return io.LimitReader(buf.Buffer, buf.Len()).Read(p)
}

func (buf *memory) ReadFrom(r io.Reader) (n int64, err error) {
	return buf.Buffer.ReadFrom(io.LimitReader(r, Gap(buf)))
}
