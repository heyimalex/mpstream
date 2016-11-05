package mpstream

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"testing"
)

func ReadAllAByteAtATime(r io.Reader) ([]byte, error) {
	buf := make([]byte, 1)
	var buf2 bytes.Buffer
	for {
		if n, err := r.Read(buf); err == io.EOF {
			buf2.Write(buf[:n])
			break
		} else if err != nil {
			return nil, err
		} else {
			buf2.Write(buf[:n])
		}
	}
	return buf2.Bytes(), nil
}

func TestBuildWithBoundary(t *testing.T) {

	const boundary = "xxxtestboundaryxxx"

	stream, err := BuildWithBoundary(boundary, []Part{
		MakeStringPart("foo", "fux"),
		MakeStringPart("bar", "yolo"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	result1, err := ReadAllAByteAtATime(stream)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.SetBoundary(boundary); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("foo", "fux"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("bar", "yolo"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	result2 := buf.Bytes()

	if !bytes.Equal(result1, result2) {
		t.Error("results were not equal")
		t.Errorf("- mpstream:\n%s", result1)
		t.Errorf("- multipart:\n%s", result2)
	}

	if mw.FormDataContentType() != stream.ContentType() {
		t.Error("content types were not equal")
		t.Errorf("- mpstream:  %s", stream.ContentType())
		t.Errorf("- multipart: %s", mw.FormDataContentType())
	}

	if stream.ContentLength() != int64(len(result1)) {
		t.Error("content length was incorrect")
		t.Errorf("- expected: %d", int64(len(result1)))
		t.Errorf("- actual:   %d", stream.ContentLength())
	}
}



func BenchmarkHello(b *testing.B) {
	const boundary = "xxxtestboundaryxxx"
    for i := 0; i < b.N; i++ {
        stream, err := BuildWithBoundary(boundary, []Part{
			MakeStringPart("foo", "fux"),
			MakeStringPart("bar", "yolo"),
			MakeStringPart("foo", "fux"),
			MakeStringPart("bar", "yolo"),
		})
		if err != nil {
			b.Fatal(err)
		}
		ioutil.ReadAll(stream)
    }
}
