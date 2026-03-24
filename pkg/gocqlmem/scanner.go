package gocqlmem

import (
	"fmt"
)

type iterScanner struct {
	Iter    *gocqlmemIter
	Cols    []any
	IsValid bool
}

func (is *iterScanner) Next() bool {
	if is.Iter.pos >= len(is.Iter.retrievedValues) {
		return false
	}
	for i := range len(is.Iter.retrievedColumnInfos) {
		is.Cols[i] = is.Iter.retrievedValues[is.Iter.pos][i]
	}
	is.Iter.pos++
	return true
}

func (is *iterScanner) Scan(dest ...any) error {
	if len(dest) < len(is.Cols) {
		return fmt.Errorf("cannot scan %d columns to dest of length %d", len(is.Cols), len(dest))
	}

	for i := range len(is.Cols) {
		if is.Cols[i] == nil {
			dest[i] = nil
		} else {
			if err := clientTypedValueToProvidedPtr(is.Cols[i], dest[i]); err != nil {
				return fmt.Errorf("cannot scan column %d: %s", i, err.Error())
			}
		}
	}
	return nil
}
func (is *iterScanner) Err() error {
	iter := is.Iter
	is.Iter = nil
	is.Cols = nil
	is.IsValid = false
	return iter.Close()
}
