package tui

import (
	"errors"
	"testing"
)

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "ordinary error", err: errors.New("DiskNotFoundError: resource not found"), want: false},
		{name: "HTTP 401", err: errors.New("HTTP 401: Unauthorized"), want: true},
		{name: "HTTP 403", err: errors.New("HTTP 403: Forbidden"), want: true},
		{name: "InvalidToken", err: errors.New("InvalidToken: the token is expired"), want: true},
		{name: "invalid_token", err: errors.New("invalid_token"), want: true},
		{name: "unauthorized lowercase", err: errors.New("unauthorized access"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthError(tt.err); got != tt.want {
				t.Errorf("isAuthError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAsFatalErrorMsg(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantNil   bool
		wantTitle string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:    "ordinary error",
			err:     errors.New("DiskNotFoundError: resource not found"),
			wantNil: true,
		},
		{
			name:    "auth error is not handled here",
			err:     errors.New("HTTP 401: Unauthorized"),
			wantNil: true,
		},
		{
			name:      "DiskAPIDisabledForOverdraftUserError",
			err:       errors.New("DiskAPIDisabledForOverdraftUserError: API недоступно"),
			wantTitle: "Storage Overdraft",
		},
		{
			name:      "overdraft lowercase",
			err:       errors.New("overdraft limit exceeded"),
			wantTitle: "Storage Overdraft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := asFatalErrorMsg(tt.err)
			if tt.wantNil {
				if got != nil {
					t.Errorf("asFatalErrorMsg(%v) = %+v, want nil", tt.err, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("asFatalErrorMsg(%v) = nil, want fatalErrorMsg with title %q", tt.err, tt.wantTitle)
			}
			if got.title != tt.wantTitle {
				t.Errorf("title = %q, want %q", got.title, tt.wantTitle)
			}
			if got.body == "" {
				t.Error("body should not be empty")
			}
			if got.hint == "" {
				t.Error("hint should not be empty")
			}
			if got.detail != tt.err {
				t.Error("detail should be the original error")
			}
		})
	}
}
