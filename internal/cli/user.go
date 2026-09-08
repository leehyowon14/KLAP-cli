package cli

import (
	"context"
	"errors"
	"fmt"
	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func (r Runner) runUser(ctx context.Context, service *app.Service, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap user <list|select|rm>")
	}

	switch args[0] {
	case "list":
		users, err := service.Users(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			_, _ = fmt.Fprintln(r.Out, "저장된 유저가 없습니다")
			return nil
		}
		for _, user := range users {
			prefix := " "
			if user.Current {
				prefix = "*"
			}
			_, _ = fmt.Fprintf(r.Out, "%s %s\n", prefix, user.User.StudentID)
		}
		return nil
	case "select":
		if len(args) != 2 {
			return errors.New("usage: klap user select <학번>")
		}
		if err := service.SelectUser(ctx, args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "현재 유저: %s\n", args[1])
		return nil
	case "rm":
		if len(args) != 2 {
			return errors.New("usage: klap user rm <학번>")
		}
		if err := service.RemoveUser(ctx, args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(r.Out, "삭제 완료: %s\n", args[1])
		return nil
	default:
		return fmt.Errorf("unknown user command: %s", args[0])
	}
}
