package mpstream

import (
	"bytes"
	//"crypto/rand"
	//"encoding/hex"
	//"errors"
	"fmt"
	"io"
	//"net/textproto"
	//"os"
	//"path/filepath"
	//"strings"
	//"mime"
)

func BuildWithBoundary2(boundary string, parts []Part) (*mpstream, int64) {


	headers := make([][]byte, len(parts))
	bodies := make([]io.Reader, len(parts))

	// Delimiter size averages out to six, and there are len(parts) + 1 delimiters.
	// head(4):   --<boundary>CRLF
	// middle(6): CRLF--<boundary>CRLF
	// tail(8):  CRLF--<boundary>--CRLF
	size := int64((len(parts) + 1) * (len(boundary) + 6))

	for i, part := range parts {

		var b bytes.Buffer
		for k, vv := range part.Header {
			for _, v := range vv {
				fmt.Fprintf(&b, "%s: %s\r\n", k, v)
			}
		}
		b.WriteString("\r\n")

		headers[i] = b.Bytes()
		size += int64(b.Len())

		bodies[i] = part.Body
		size += part.Size
	}

	streamer := &mpstream{
		mode: mBoundary,
		index: 0,
		offset: 2, // The head delimiter is conveniently just delimiter[2:]
		delimiter: []byte("\r\n--" + boundary + "\r\n"),
		headers: headers,
		bodies: bodies,
		trailer: []byte("\r\n--" + boundary + "--\r\n"),
	}
	return streamer, size
}

type readmode int
const (
	mBoundary = iota
	mHeader
	mBody
	mTrailer
)

type mpstream struct {
	mode readmode
	index int
	offset int

	//boundary string

	delimiter      []byte
	headers        [][]byte
	bodies         []io.Reader
	trailer        []byte

	// todo: check if size was accurate
	// todo: aggregate closing errors
}

func (s *mpstream) Read(p []byte) (n int, err error) {

	var (
		delimiter []byte   = s.delimiter
		index     int      = s.index
		offset    int      = s.offset
		header    []byte
		nn        int
	)

	switch s.mode {
	case mBoundary:
		goto LBOUNDARY
	case mHeader:
		goto LHEADER
	case mBody:
		goto LBODY
	case mTrailer:
		goto LTRAILER
	}

LBOUNDARY:
	nn = copy(p[n:], delimiter[offset:])
	offset += nn
	n += nn

	if n == len(p) { // Write complete
		if offset != len(delimiter) { // Read incomplete
			s.index = index
			s.offset = offset
			s.mode = mBoundary
		} else { // Read complete
			s.index = index
			s.offset = 0
			s.mode = mHeader
		}
		return n, nil
	} else { // Read complete
		offset = 0
		goto LHEADER
	}

LHEADER:
	header = s.headers[index]
	nn = copy(p[n:], header[offset:])
	offset += nn
	n += nn

	if n == len(p) { // Write complete
		if offset != len(header) { // Read incomplete
			s.index = index
			s.offset = offset
			s.mode = mHeader
		} else { // Read complete
			s.index = index
			s.offset = 0
			s.mode = mBody
		}
		return n, nil
	} else { // Read complete
		offset = 0
		goto LBODY
	}

LBODY:
	nn, err = s.bodies[index].Read(p[n:])
	n += nn

	if err != nil && err != io.EOF {
		return
	}

	if n == len(p) {
		if err == nil {
			s.index = index
			s.offset = 0
			s.mode = mBody
			return n, nil
		}
		index += 1
		s.index = index
		s.offset = 0
		if len(s.bodies) == index {
			s.mode = mTrailer
		} else {
			s.mode = mBoundary
		}
		return n, nil
	} else {
		index += 1
		if len(s.bodies) == index {
			s.mode = mTrailer
			goto LTRAILER
		} else {
			goto LBOUNDARY
		}
	}

LTRAILER:
	nn = copy(p[n:], s.trailer[offset:])
	offset += nn
	n += nn
	s.offset = offset
	if offset == len(s.trailer) {
		return n, io.EOF
	} else {
		return n, nil
	}
}

func (s *mpstream) Close() error {
	var errs []error
	for _, body := range s.bodies {
		if c, ok := body.(io.Closer); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return &multiCloseError{errs}
	}
}

type multiCloseError struct {
	errs []error
}

func (e *multiCloseError) Error() string {
	errs := e.errs
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "mpstream: %d errors while closing parts: ", len(errs))
	fmt.Fprintf(&msg, "[%d]: %s", 0, errs[0])
	for i := 1; i < len(errs); i++ {
		fmt.Fprintf(&msg, ", [%d]: %s", i, errs[i])
	}
	return msg.String()
}

func (e *multiCloseError) WrappedErrors() []error {
	return e.errs
}
