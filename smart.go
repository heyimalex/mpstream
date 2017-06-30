package mpstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type PartBuilder func() (Part, error)

func StringPart(fieldname, value string) PartBuilder {
	return func() (Part, error) {
		return FormField(fieldname, []byte(value)), nil
	}
}

func BytePart(fieldname string, value []byte) PartBuilder {
	return func() (Part, error) {
		return FormField(fieldname, value), nil
	}
}

func JSONPart(fieldname string, value interface{}) PartBuilder {
	return func() (Part, error) {
		encoded, err := json.Marshal(value)
		if err != nil {
			return Part{}, fmt.Errorf(
				`error marshalling field "%s" to json: %s`, fieldname, err,
			)
		}
		return FormField(fieldname, encoded), nil
	}
}

func FilePart(fieldname string, filename string) PartBuilder {
	return func() (Part, error) {
		p, _, err := FormFile(fieldname, filename)
		if err != nil {
			return Part{}, fmt.Errorf(
				`error creating file part for field "%s": %s`, fieldname, err,
			)
		}
		return p, nil
	}
}

type SmartStream struct {
	Stream
	parts []Part
}

func NewSmart(builders ...PartBuilder) (*SmartStream, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	return NewSmartWithBoundary(boundary, builders...)
}

func NewSmartWithBoundary(boundary string, builders ...PartBuilder) (*SmartStream, error) {
	parts := make([]Part, len(builders))
	for i, build := range builders {
		var err error
		parts[i], err = build()
		if err != nil {
			closeparts(parts) // swallows error
			return nil, fmt.Errorf(
				"mpstream: error building part[%d]: %s", i, err,
			)
		}
	}

	stream, err := NewWithBoundary(boundary, parts...)
	if err != nil {
		closeparts(parts) // swallows error
		return nil, err
	}

	return &SmartStream{
		Stream: *stream,
		parts:  parts,
	}, nil
}

func (s *SmartStream) Close() error {
	return closeparts(s.parts)
}

func closeparts(parts []Part) error {
	var errs []error
	for _, p := range parts {
		if c, ok := p.Body.(io.Closer); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		return nil
	} else {
		return multiCloseErr{errs}
	}
}

type multiCloseErr struct {
	errs []error
}

func (e multiCloseErr) Error() string {
	errs := e.errs
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "mpstream: %d errors while closing parts: ", len(errs))
	fmt.Fprintf(&msg, "[%d]: %s", 0, errs[0])
	for i := 1; i < len(errs); i++ {
		fmt.Fprintf(&msg, ", [%d]: %s", i, errs[i])
	}
	return msg.String()
}

func (e multiCloseErr) WrappedErrors() []error {
	return e.errs
}
