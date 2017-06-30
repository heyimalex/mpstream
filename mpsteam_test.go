package mpstream

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"testing"
)

func TestNewWithBoundary(t *testing.T) {

	const boundary = "xxxtestboundaryxxx"

	stream, err := NewWithBoundary(boundary,
		FormField("foo", []byte("fux")),
		FormField("bar", []byte("yolo")),
	)
	if err != nil {
		t.Fatal(err)
	}
	result1, err := ioutil.ReadAll(stream)
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
