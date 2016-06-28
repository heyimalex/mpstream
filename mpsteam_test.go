package mpstream

import (
    "bytes"
    "io"
    "mime/multipart"
    "testing"
)

func TestSize(t *testing.T) {
    s := New([]*Part{
        NewFormField("foo").SetContentFromString("fux"),
        NewFormField("bar").SetContentFromString("yolo"),
    })

    var b bytes.Buffer
    n, err := io.Copy(&b, s)
    if err != nil {
        t.Fatal(err)
    }
    if int64(n) != s.ContentLength() {
        t.Fatal("%d != %d", n, s.ContentLength())
    }
}

func TestEQ(t *testing.T) {
    s := New([]*Part{
        NewFormField("foo").SetContentFromString("fux"),
        NewFormField("bar").SetContentFromString("yolo"),
    })
    var b1 bytes.Buffer
    _, err := io.Copy(&b1, s)
    if err != nil {
        t.Fatal(err)
    }

    var b2 bytes.Buffer
    mw := multipart.NewWriter(&b2)
    if err := mw.SetBoundary(s.Boundary()); err != nil {
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

    if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
        t.Fatal("fuck, they're not Equal")
    }

    if mw.FormDataContentType() != s.ContentType() {
        t.Fatal("content type not equal")
    }
}
