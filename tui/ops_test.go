package tui

import (
	"testing"

	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/i18n"
)

func TestNewAPIError(t *testing.T) {
	both := &disk.ErrorResponse{Error: "DiskNotFoundError",
		Message: "Не удалось найти запрошенный ресурс.", Description: "Resource not found."}
	tests := []struct {
		name  string
		lang  i18n.Lang
		input *disk.ErrorResponse
		want  string // empty string means nil error expected
	}{
		{"nil input", i18n.English, nil, ""},
		{"English interface shows the English description", i18n.English, both, "Resource not found."},
		{"Russian interface shows the Russian message", i18n.Russian, both, "Не удалось найти запрошенный ресурс."},
		{"falls back to the other language", i18n.English, &disk.ErrorResponse{Message: "Доступ запрещён."}, "Доступ запрещён."},
		{"falls back to the error code", i18n.Russian, &disk.ErrorResponse{Error: "UnauthorizedError"}, "UnauthorizedError"},
		{"nothing at all", i18n.English, &disk.ErrorResponse{}, "unknown error"},
	}
	defer i18n.Set(i18n.English)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i18n.Set(tt.lang)
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
