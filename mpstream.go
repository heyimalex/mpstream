package mpstream

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"mime"
)

type Part struct {
	Header textproto.MIMEHeader
	Size   int64
	Body   io.Reader
}

type Stream struct {
	reader   io.Reader
	boundary string
	size     int64
	parts    []Part
}

func Build(parts []Part) (*Stream, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	return BuildWithBoundary(boundary, parts)
}

func BuildWithBoundary(boundary string, parts []Part) (*Stream, error) {
	if err := validateBoundary(boundary); err != nil {
		return nil, err
	}

	reader, size := BuildWithBoundary2(boundary, parts)
	/*readers := make([]io.Reader, len(parts)*3+1)

	// Delimiter size averages out to six, and there are len(parts) + 1 delimiters.
	// head(4):   --<boundary>CRLF
	// middle(6): CRLF--<boundary>CRLF
	// close(8):  CRLF--<boundary>--CRLF
	size := int64((len(parts) + 1) * (len(boundary) + 6))

	delimiter := []byte("\r\n--" + boundary + "\r\n")
	headDelimiter := delimiter[2:]
	closeDelimiter := []byte("\r\n--" + boundary + "--\r\n")

	readers[0] = bytes.NewReader(headDelimiter)
	readers[len(parts)*3] = bytes.NewReader(closeDelimiter)
	for i := 1; i < len(parts); i++ {
		readers[i*3] = bytes.NewReader(delimiter)
	}

	for i, part := range parts {

		var b bytes.Buffer
		for k, vv := range part.Header {
			for _, v := range vv {
				fmt.Fprintf(&b, "%s: %s\r\n", k, v)
			}
		}
		b.WriteString("\r\n")

		readers[i*3+1] = &b
		size += int64(b.Len())

		readers[i*3+2] = part.Body
		size += part.Size
	}*/

	streamer := &Stream{
		reader:   reader,
		boundary: boundary,
		size:     size,
		parts:    parts,
	}
	return streamer, nil
}

func (s *Stream) ContentType() string {
	return "multipart/form-data; boundary=" + s.boundary
}

func (s *Stream) ContentLength() int64 {
	return s.size
}

func (s *Stream) Boundary() string {
	return s.boundary
}

func (s *Stream) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *Stream) Close() error {
	var errs []error
	for _, part := range s.parts {
		if c, ok := part.Body.(io.Closer); ok {
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
		return &streamCloseError{errs}
	}
}

type streamCloseError struct {
	errs []error
}

func (e *streamCloseError) Error() string {
	errs := e.errs
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "mpstream: encountered %d errors while closing parts:", len(errs))
	fmt.Fprintf(&msg, "[%d]: %s", 0, errs[0])
	for i := 1; i < len(errs); i++ {
		fmt.Fprintf(&msg, ", [%d]: %s", i, errs[i])
	}
	return msg.String()
}

func randomBoundary() (string, error) {
	var buf [30]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func validateBoundary(boundary string) error {
	// rfc2046#section-5.1.1
	if len(boundary) < 1 || len(boundary) > 69 {
		return errors.New("mpstream: invalid boundary length")
	}
	for _, b := range boundary {
		if 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' {
			continue
		}
		switch b {
		case '\'', '(', ')', '+', '_', ',', '-', '.', '/', ':', '=', '?':
			continue
		}
		return errors.New("mpstream: invalid boundary character")
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func MakeBytePart(fieldname string, body []byte) Part {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	return Part{
		Header: h,
		Size:   int64(len(body)),
		Body:   bytes.NewReader(body),
	}
}

func MakeStringPart(fieldname, body string) Part {
	return MakeBytePart(fieldname, []byte(body))
}

func MakeFilePart(fieldname, filename string) (p Part, err error) {
	stats, err := os.Stat(filename)
	if err != nil {
		return
	}
	if stats.IsDir() {
		err = errors.New("mpstream: cannot make file part for directory")
		return
	}
	p.Size = stats.Size()
	name := stats.Name()

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType != "" {
		contentType = "application/octet-stream"
	}
	h.Set("Content-Type", contentType)

	p.Header = h
	p.Body = &lazyFile{filename: filename}
	return

}

type lazyFile struct {
	filename string
	file     *os.File
}

func (lf *lazyFile) Read(p []byte) (n int, err error) {
	if lf.file == nil {
		lf.file, err = os.Open(lf.filename)
		if err != nil {
			return 0, fmt.Errorf("mpstream: %s", err)
		}
	}
	return lf.file.Read(p)
}

func (lf *lazyFile) Close() error {
	if lf.file != nil {
		return lf.file.Close()
	}
	return nil
}
