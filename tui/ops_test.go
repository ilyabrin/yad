package tui

import (
	"testing"

	"github.com/ilyabrin/disk"
)

func TestNewAPIError(t *testing.T) {
	tests := []struct {
		name  string
		input *disk.ErrorResponse
		want  string // empty string means nil error expected
	}{
		{
			name:  "nil input",
			input: nil,
			want:  "",
		},
		{
			name:  "message only",
			input: &disk.ErrorResponse{Message: "not found"},
			want:  "not found",
		},
		{
			name:  "error field used when message is empty",
			input: &disk.ErrorResponse{Error: "DiskNotFoundError"},
			want:  "DiskNotFoundError",
		},
		{
			name:  "message takes priority over error field",
			input: &disk.ErrorResponse{Error: "DiskNotFoundError", Message: "Resource not found"},
			want:  "Resource not found",
		},
		{
			name:  "description appended",
			input: &disk.ErrorResponse{Message: "Forbidden", Description: "token expired"},
			want:  "Forbidden: token expired",
		},
		{
			name:  "error field with description",
			input: &disk.ErrorResponse{Error: "UnauthorizedError", Description: "invalid token"},
			want:  "UnauthorizedError: invalid token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newAPIError(tt.input)
			if tt.want == "" {
				if err != nil {
					t.Errorf("newAPIError(%v) = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("newAPIError(%v) = nil, want %q", tt.input, tt.want)
			}
			if got := err.Error(); got != tt.want {
				t.Errorf("newAPIError(%v).Error() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
