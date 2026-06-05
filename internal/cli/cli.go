package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/kw-klap/klap-cli/internal/account"
	"github.com/kw-klap/klap-cli/internal/klas"
	"github.com/kw-klap/klap-cli/internal/ui"
)

func Run(ctx context.Context, args []string) error {
	store, err := account.NewStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "auth":
		return runAuth(ctx, store)
	case "user":
		return runUser(ctx, store, args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runAuth(ctx context.Context, store *account.Store) error {
	credentials, err := ui.RunAuthForm()
	if err != nil {
		return err
	}
	if credentials.StudentID == "" || credentials.Password == "" {
		return errors.New("학번과 비밀번호를 모두 입력해야 합니다")
	}

	client, err := klas.NewClient()
	if err != nil {
		return err
	}

	session, err := client.Login(ctx, credentials.StudentID, credentials.Password)
	if err != nil {
		return fmt.Errorf("로그인 검증 실패: %w", err)
	}

	if err := store.Save(ctx, credentials.StudentID, credentials.Password, session); err != nil {
		return fmt.Errorf("계정 저장 실패: %w", err)
	}

	fmt.Printf("저장 완료: %s\n", credentials.StudentID)
	return nil
}

func runUser(ctx context.Context, store *account.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap user <list|rm>")
	}

	switch args[0] {
	case "list":
		users, err := store.List(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("저장된 유저가 없습니다")
			return nil
		}
		for _, user := range users {
			fmt.Println(user.StudentID)
		}
		return nil
	case "rm":
		if len(args) != 2 {
			return errors.New("usage: klap user rm <학번>")
		}
		if err := store.Remove(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("삭제 완료: %s\n", args[1])
		return nil
	default:
		return fmt.Errorf("unknown user command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println(`KLAP CLI

Usage:
  klap auth              KLAS 로그인 검증 후 계정 저장
  klap user list         저장된 학번 목록 출력
  klap user rm <학번>    저장된 계정 삭제`)
}
