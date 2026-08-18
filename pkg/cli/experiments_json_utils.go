package cli

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

func parsePagedJSONArray[T any](output string) ([]T, error) {
	var result []T
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var page []T
		if err := decoder.Decode(&page); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		result = append(result, page...)
	}
	return result, nil
}
