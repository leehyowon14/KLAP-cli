package macos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

func runJSONBridge(ctx context.Context, path, label string, request any, result any) error {
	var input []byte
	if request != nil {
		var err error
		input, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("%s payload 직렬화 실패: %w", label, err)
		}
	}
	err := (ProcessRunner{}).Run(ctx, bridgeSpec(path, input), func(stdout io.Reader) error {
		body, err := io.ReadAll(stdout)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("Swift %s bridge 응답 파싱 실패: %w", label, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Swift %s bridge 실패: %w", label, err)
	}
	return nil
}
