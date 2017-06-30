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
	"strings"
)

type Part struct {
	Header textproto.MIMEHeader
	Size   int64
	Body   io.Reader
}

func New(parts ...Part) (*Stream, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	return NewWithBoundary(boundary, parts...)
}

func NewWithBoundary(boundary string, parts ...Part) (*Stream, error) {
	if err := validateBoundary(boundary); err != nil {
		return nil, err
	}

	if len(parts) == 0 {
		return nil, errors.New("mpstream: must pass at least one part")
	}

	readers := make([]io.Reader, len(parts)*3+1)

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
	}

	streamer := &Stream{
		Reader:   io.MultiReader(readers...),
		boundary: boundary,
		size:     size,
	}
	return streamer, nil
}

type Stream struct {
	io.Reader
	boundary string
	size     int64
}

func (s Stream) ContentType() string {
	return "multipart/form-data; boundary=" + s.boundary
}

func (s Stream) ContentLength() int64 {
	return s.size
}

func (s Stream) Boundary() string {
	return s.boundary
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

func FormField(fieldname string, value []byte) Part {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	return Part{
		Header: h,
		Size:   int64(len(value)),
		Body:   bytes.NewReader(value),
	}
}

func FormFile(fieldname, filename string) (p Part, f *os.File, err error) {
	stats, err := os.Stat(filename)
	if err != nil {
		return
	}
	if stats.IsDir() {
		err = fmt.Errorf(
			`mpstream: error creating file part "%s": path "%s" is a directory`,
			fieldname, filename,
		)
		return
	}
	p.Size = stats.Size()

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(stats.Name())))
	h.Set("Content-Type", "application/octet-stream")
	p.Header = h

	f, err = os.Open(filename)
	if err != nil {
		return
	}

	p.Body = f
	return
}
