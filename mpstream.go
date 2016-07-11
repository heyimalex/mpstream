package mpstream

import (
    "bytes"
    "crypto/rand"
    "fmt"
    "io"
    "net/textproto"
    "os"
    "strings"
)

type Streamer struct {
    io.Reader
    boundary string
    size int64
}

func New(parts []*Part) *Streamer {
    boundary := randomBoundary()
    readers := make([]io.Reader, len(parts) * 3 + 1)
    delimiter := []byte(fmt.Sprintf("\r\n--%s\r\n", boundary))

    // Delimiter size averages out to six, and there are len(parts) + 1 delimiters.
    // head(4):   --<boundary>CRLF
    // middle(6): CRLF--<boundary>CRLF
    // close(8):  CRLF--<boundary>--CRLF
    size := int64((len(parts) + 1) * (len(boundary) + 6))

    for i, part := range parts {

        if i == 0 {
            readers[i*3] = bytes.NewReader(delimiter[2:])
        } else {
            readers[i*3] = bytes.NewReader(delimiter)
        }

        var b bytes.Buffer
        for k, vv := range part.Header {
            for _, v := range vv {
                fmt.Fprintf(&b, "%s: %s\r\n", k, v)
            }
        }
        fmt.Fprintf(&b, "\r\n")

        readers[i*3 + 1] = &b
        size += int64(b.Len())

        readers[i*3 + 2] = part.Content
        size += part.Size
    }

    closeDelimiter := fmt.Sprintf("\r\n--%s--\r\n", boundary)
    readers[len(parts) * 3] = bytes.NewReader([]byte(closeDelimiter))

    return &Streamer{
        Reader: io.MultiReader(readers...),
        boundary: boundary,
        size: size,
    }
}

func (s *Streamer) ContentType() string {
    return "multipart/form-data; boundary=" + s.boundary
}

func (s *Streamer) ContentLength() int64 {
    return s.size
}

func (s *Streamer) Boundary() string {
    return s.boundary
}

func randomBoundary() string {
    var buf [30]byte
    _, err := rand.Read(buf[:])
    if err != nil {
        panic(err)
    }
    return fmt.Sprintf("%x", buf[:])
}

type Part struct {
    Header  textproto.MIMEHeader
    Size    int64
    Content io.Reader
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
    return quoteEscaper.Replace(s)
}

func NewFormFile(fieldname, filename string) *Part {
    h := make(textproto.MIMEHeader)
    h.Set("Content-Disposition",
        fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
            escapeQuotes(fieldname), escapeQuotes(filename)))
    h.Set("Content-Type", "application/octet-stream")
    return &Part{ Header: h }
}

func NewFormField(fieldname string) *Part {
    h := make(textproto.MIMEHeader)
    h.Set("Content-Disposition",
        fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
    return &Part{ Header: h }
}

func (p *Part) SetContentFromBytes(b []byte) *Part {
    p.Size = int64(len(b))
    p.Content = bytes.NewReader(b)
    return p
}

func (p *Part) SetContentFromString(s string) *Part {
    return p.SetContentFromBytes([]byte(s))
}

func (p *Part) SetContentFromFile(f *os.File) error {
    info, err := f.Stat()
    if err != nil {
        return err
    }
    p.Size = info.Size()
    p.Content = f
    return nil
}
