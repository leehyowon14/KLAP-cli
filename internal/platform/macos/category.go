package macos

import (
	"context"
	"errors"
	"github.com/leehyowon14/KLAP-cli/internal/category"
	"runtime"
)

type CategoryBridge struct{ path string }

func NewCategoryBridge(path string) CategoryBridge { return CategoryBridge{path: path} }
func (b CategoryBridge) List(ctx context.Context) (category.Options, error) {
	if runtime.GOOS != "darwin" {
		return category.Options{}, errors.New("category list는 현재 macOS에서만 지원합니다")
	}
	var result category.Options
	if err := runJSONBridge(ctx, b.path, "category", nil, &result); err != nil {
		return category.Options{}, err
	}
	return result, nil
}
