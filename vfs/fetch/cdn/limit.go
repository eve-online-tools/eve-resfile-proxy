package cdn

import (
	"fmt"
	"io"
)

func readLimitedBody(r io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(r, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("response exceeds max size (%d bytes)", max)
	}
	return data, nil
}
