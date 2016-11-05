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
		offset: 2,
		delimiter: []byte("\r\n--" + boundary + "\r\n"),
		headers: headers,
		bodies: bodies,
		tailDelimiter: []byte("\r\n--" + boundary + "--\r\n"),
	}
	return streamer, size
}

// modes: boundary, header, body, end

type readmode int
const (
	mBoundary = iota
	mHeader
	mBody
)

type mpstream struct {
	mode readmode
	index int
	offset int

	//boundary string

	delimiter      []byte
	headers        [][]byte
	bodies         []io.Reader
	tailDelimiter  []byte

	// todo: check if size was accurate
	// todo: aggregate closing errors
}

func (s *mpstream) Read(p []byte) (n int, err error) {

	for s.index < len(s.bodies) {

		if s.mode == mBoundary {
			nn := copy(p[n:], s.delimiter[s.offset:])
			s.offset += nn
			n += nn
			if s.offset == len(s.delimiter) {
				s.offset = 0
				s.mode = mHeader
			}
			if n == len(p) {
				return n, nil
			}
		}

		if s.mode == mHeader {
			header := s.headers[s.index]
			nn := copy(p[n:], header[s.offset:])
			s.offset += nn
			n += nn
			if s.offset == len(header) {
				s.offset = 0
				s.mode = mBody
			}
			if n == len(p) {
				return n, nil
			}
		}

		if s.mode == mBody {
			body := s.bodies[s.index]
			nn, err := body.Read(p[n:])
			n += nn
			if err != nil {
				if err != io.EOF {
					return n, err
				}
				s.index += 1
				s.mode = mBoundary
			}
			if n == len(p) {
				return n, nil
			}
		}
	}

	nn := copy(p[n:], s.tailDelimiter[s.offset:])
	s.offset += nn
	n += nn
	if s.offset == len(s.tailDelimiter) {
		return n, io.EOF
	}
	//if n == len(p) {
	return n, nil
	//}
}

func (s *mpstream) Close() error {
	return nil
}
