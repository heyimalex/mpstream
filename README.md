# mpstream

Small library for writing multipart requests.

The stdlib `"mime/multipart"` package has a clunky api for creating requests. The main issue is that requests take a reader, but `mime/multipart` provides a writer. The burden of managing between those two interfaces falls on the user. This package provides a nicer api.

### Usage

```go

// mime/multipart
func stdlibVersion(fieldname, filename string) (*http.Request, error) {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    var buf bytes.Buffer
    w := multipart.NewWriter(&buf)
    fw, err := w.CreateFormFile(fieldname, filepath.Base(filename))
    if err != nil {
        return err
    }

    if _, err := io.Copy(fw, file); err != nil {
        return nil, err
    }

    if err := w.Close(); err != nil {
        return nil, err
    }

    req, err := http.NewRequest(
        "POST",
        "http://www.example.com",
        &buf,
    )
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", w.FormDataContentType())
    req.ContentLength = buf.Len()
    return req, nil
}

func mpstreamVersion(fieldname, filename string) (*http.Request, error) {
    filePart, file, err := mpstream.FormFile(fieldname, filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    ms, err := mpstream.New(filePart)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequest(
        "POST",
        "http://www.example.com",
        ms,
    )
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", ms.ContentType())
    req.ContentLength = ms.ContentLength()
    return req, nil
}

func mpstreamSmart(fieldname, filename)  (*http.Request, error) {
    ms, err := mpstream.NewSmart(mpstream.FilePart(fieldname, filename))
    if err != nil {
        return nil, err
    }
    defer ms.Close()

    req, err := http.NewRequest(
        "POST",
        "http://www.example.com",
        ms,
    )
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", ms.ContentType())
    req.ContentLength = ms.ContentLength()
    return req, nil
}

```
