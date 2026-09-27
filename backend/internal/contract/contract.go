package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"io"
)

type Contract struct{ Spec *openapi3.T }

func Load(path string) (*Contract, error) {
	l := openapi3.NewLoader()
	s, err := l.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	// The supplied v1.2 spec attaches descriptions to $ref. Permit only this
	// documentation sibling; validation of request bodies remains strict.
	if err = s.Validate(context.Background(), openapi3.AllowExtraSiblingFields("description")); err != nil {
		return nil, err
	}
	return &Contract{s}, nil
}

// Reject duplicate keys at every depth before the decoder can overwrite them.
func UniqueJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := walk(dec); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return fmt.Errorf("JSON contains trailing content")
	}
	return nil
}
func walk(dec *json.Decoder) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	v, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch v {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				return err
			}
			name, ok := k.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate JSON key")
			}
			seen[name] = true
			if err := walk(dec); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			if err := walk(dec); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid JSON delimiter")
	}
	_, err = dec.Token()
	return err
}
func (c *Contract) Validate(name string, b []byte) error {
	if err := UniqueJSON(b); err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	s, ok := c.Spec.Components.Schemas[name]
	if !ok {
		return fmt.Errorf("unknown schema %s", name)
	}
	return s.Value.VisitJSON(value)
}
