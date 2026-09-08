package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/leehyowon14/KLAP-cli/internal/app"
)

func TestRunSelectsPresentation(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		tui  bool
	}{
		{"empty", nil, true}, {"explicit TUI", []string{"tui"}, true},
		{"TUI arguments", []string{"tui", "extra"}, true},
		{"help", []string{"--help"}, false}, {"command", []string{"dashboard"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &app.Service{}
			wantErr := errors.New("presentation failed")
			created, cliCalls, tuiCalls := 0, 0, 0
			ctx := context.Background()
			err := runWithFactory(ctx, tc.args, func() (*app.Service, error) { created++; return service, nil },
				func(gotCtx context.Context, args []string, s *app.Service) error {
					cliCalls++
					if gotCtx != ctx || s != service || !reflect.DeepEqual(args, tc.args) {
						t.Fatal("CLI dependencies changed")
					}
					return wantErr
				},
				func(gotCtx context.Context, s *app.Service) error {
					tuiCalls++
					if gotCtx != ctx || s != service {
						t.Fatal("TUI dependencies changed")
					}
					return wantErr
				})
			if !errors.Is(err, wantErr) || created != 1 || cliCalls+tuiCalls != 1 || (tuiCalls == 1) != tc.tui {
				t.Fatalf("error=%v created=%d cli=%d tui=%d", err, created, cliCalls, tuiCalls)
			}
		})
	}
}

func TestRunPropagatesInitializationFailure(t *testing.T) {
	wantErr := errors.New("initialization failed")
	err := runWithFactory(context.Background(), []string{"help"}, func() (*app.Service, error) { return nil, wantErr },
		func(context.Context, []string, *app.Service) error {
			t.Fatal("CLI ran after initialization failure")
			return nil
		},
		func(context.Context, *app.Service) error { t.Fatal("TUI ran after initialization failure"); return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v", err)
	}
}
