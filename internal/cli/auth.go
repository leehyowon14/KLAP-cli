package cli

import (
	"context"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
	"github.com/leehyowon14/KLAP-cli/internal/ui"
)

func (r Runner) runAuth(ctx context.Context, service *app.Service) error {
	credentials, err := ui.RunAuthForm()
	if err != nil {
		return err
	}
	if err := service.Authenticate(ctx, credentials.StudentID, credentials.Password); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(r.Out, "저장 완료: %s\n", credentials.StudentID)
	return nil
}
