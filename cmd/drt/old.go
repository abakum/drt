//go:build ignore

package main

import (
	"bytes"
	"encoding/gob"
)

func copyMap(in, out interface{}) {
	buf := new(bytes.Buffer)
	gob.NewEncoder(buf).Encode(in)
	gob.NewDecoder(buf).Decode(out)
}
