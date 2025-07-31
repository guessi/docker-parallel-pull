package security

import (
	"testing"
)

func TestValidateImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		wantError bool
	}{
		{
			name:      "valid image name",
			imageName: "alpine:latest",
			wantError: false,
		},
		{
			name:      "valid image with registry",
			imageName: "docker.io/library/alpine:latest",
			wantError: false,
		},
		{
			name:      "empty image name",
			imageName: "",
			wantError: true,
		},
		{
			name:      "image name with suspicious characters",
			imageName: "alpine:latest; rm -rf /",
			wantError: true,
		},
		{
			name:      "image name too long",
			imageName: string(make([]byte, 300)),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageName(tt.imageName)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateImageName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		{
			name:     "nil error",
			input:    nil,
			expected: "",
		},
		{
			name:     "error with path",
			input:    &testError{msg: "failed to read /home/user/secret.txt"},
			expected: "failed to read [PATH_REDACTED]",
		},
		{
			name:     "error with IP",
			input:    &testError{msg: "connection failed to 192.168.1.1"},
			expected: "connection failed to [IP_REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeErrorMessage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
