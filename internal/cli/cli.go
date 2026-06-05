package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	case "course":
		return runCourse(ctx, store, args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runCourse(ctx context.Context, store *account.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: klap course list [--user <학번>]")
	}

	switch args[0] {
	case "list":
		studentID, err := selectedStudentID(ctx, store, args[1:])
		if err != nil {
			return err
		}

		client, err := authenticatedClient(ctx, store, studentID)
		if err != nil {
			return err
		}

		terms, err := client.Courses(ctx)
		if err != nil {
			refreshedClient, refreshed, refreshErr := refreshedClientAfterSessionError(ctx, store, studentID, err)
			if refreshErr != nil {
				return refreshErr
			}
			if refreshed {
				terms, err = refreshedClient.Courses(ctx)
			}
		}
		if err != nil {
			return err
		}
		printCourseList(terms)
		return nil
	default:
		return fmt.Errorf("unknown course command: %s", args[0])
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
		currentStudentID, err := store.Current(ctx)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			fmt.Println("저장된 유저가 없습니다")
			return nil
		}
		for _, user := range users {
			prefix := " "
			if user.StudentID == currentStudentID {
				prefix = "*"
			}
			fmt.Printf("%s %s\n", prefix, user.StudentID)
		}
		return nil
	case "select":
		if len(args) != 2 {
			return errors.New("usage: klap user select <학번>")
		}
		if err := store.Select(ctx, args[1]); err != nil {
			return err
		}
		fmt.Printf("현재 유저: %s\n", args[1])
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
  klap user select <학번> 현재 유저 선택
  klap user rm <학번>    저장된 계정 삭제
  klap course list       최신 학기 수업 목록 출력`)
}

func selectedStudentID(ctx context.Context, store *account.Store, args []string) (string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] != "--user" {
			continue
		}
		if i+1 >= len(args) {
			return "", errors.New("--user에는 학번이 필요합니다")
		}
		return args[i+1], nil
	}

	currentStudentID, err := store.Current(ctx)
	if err != nil {
		return "", err
	}
	if currentStudentID != "" {
		return currentStudentID, nil
	}

	users, err := store.List(ctx)
	if err != nil {
		return "", err
	}
	switch len(users) {
	case 0:
		return "", errors.New("저장된 유저가 없습니다. 먼저 klap auth를 실행하세요")
	case 1:
		return users[0].StudentID, nil
	default:
		return "", errors.New("저장된 유저가 여러 명입니다. --user <학번>을 지정하세요")
	}
}

func authenticatedClient(ctx context.Context, store *account.Store, studentID string) (*klas.Client, error) {
	client, err := klas.NewClient()
	if err != nil {
		return nil, err
	}

	session, err := store.LoadSession(ctx, studentID)
	if err == nil {
		client.SetSession(session)
		return client, nil
	}

	password, err := store.LoadPassword(ctx, studentID)
	if err != nil {
		return nil, err
	}
	session, err = client.Login(ctx, studentID, password)
	if err != nil {
		return nil, fmt.Errorf("재로그인 실패: %w", err)
	}
	if err := store.SaveSession(ctx, studentID, session); err != nil {
		return nil, err
	}
	return client, nil
}

func refreshedClientAfterSessionError(ctx context.Context, store *account.Store, studentID string, err error) (*klas.Client, bool, error) {
	if !errors.Is(err, klas.ErrSessionExpired) {
		return nil, false, err
	}

	client, clientErr := klas.NewClient()
	if clientErr != nil {
		return nil, true, clientErr
	}
	password, passwordErr := store.LoadPassword(ctx, studentID)
	if passwordErr != nil {
		return nil, true, passwordErr
	}
	session, loginErr := client.Login(ctx, studentID, password)
	if loginErr != nil {
		return nil, true, fmt.Errorf("재로그인 실패: %w", loginErr)
	}
	if saveErr := store.SaveSession(ctx, studentID, session); saveErr != nil {
		return nil, true, saveErr
	}
	return client, true, nil
}

func printCourseList(terms []klas.Term) {
	if len(terms) == 0 {
		fmt.Println("수강 학기가 없습니다")
		return
	}

	term := terms[0]
	fmt.Printf("%s (%s)\n", term.Label, term.Value)
	if len(term.Courses) == 0 {
		fmt.Println("수업이 없습니다")
		return
	}

	for index, course := range term.Courses {
		fmt.Printf("%d. %s\n", index+1, strings.TrimSpace(course.Name))
	}
}
