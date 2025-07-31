package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Security constants
const (
	MaxFileSize        = 10 * 1024 * 1024  // 10MB max file size
	MaxImages          = 1000              // Maximum number of images
	AllowedConfigPaths = "/tmp,/var/tmp,." // Allowed config file paths
)

// Security validation regex patterns
var (
	validImageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9]$`)
	validTagRegex       = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	sensitiveDataRegex  = regexp.MustCompile(`(?i)(password|token|key|secret)=[a-zA-Z0-9]+`)
	pathRegex           = regexp.MustCompile(`/[a-zA-Z0-9/_.-]+`)
	ipRegex             = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
)

// ValidateFilePath ensures the file path is safe and within allowed directories
func ValidateFilePath(filePath string) error {
	cleanPath := filepath.Clean(filePath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal detected in: %s", SanitizeLogMessage(filePath))
	}

	allowedPaths := strings.Split(AllowedConfigPaths, ",")
	allowed := false
	for _, allowedPath := range allowedPaths {
		allowedAbs, err := filepath.Abs(strings.TrimSpace(allowedPath))
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file path not in allowed directories: %s", SanitizeLogMessage(absPath))
	}

	return nil
}

// ValidateImageName validates Docker image names for security
func ValidateImageName(imageName string) error {
	if len(imageName) == 0 {
		return fmt.Errorf("image name cannot be empty")
	}

	if len(imageName) > 255 {
		return fmt.Errorf("image name too long: %d characters", len(imageName))
	}

	if strings.ContainsAny(imageName, "$`;&|<>(){}[]") {
		return fmt.Errorf("image name contains suspicious characters: %s", SanitizeLogMessage(imageName))
	}

	// Split by colon to separate name and tag
	nameTag := strings.Split(imageName, ":")
	imageParts := nameTag[0] // The part before the colon (or the whole string if no colon)

	parts := strings.Split(imageParts, "/")
	if len(parts) > 3 {
		return fmt.Errorf("image name has too many path components: %s", SanitizeLogMessage(imageName))
	}

	for _, part := range parts {
		if !validImageNameRegex.MatchString(part) {
			return fmt.Errorf("invalid image name component: %s", SanitizeLogMessage(part))
		}
	}

	// Validate tag if present
	if len(nameTag) == 2 {
		if !validTagRegex.MatchString(nameTag[1]) {
			return fmt.Errorf("invalid image tag: %s", SanitizeLogMessage(nameTag[1]))
		}
	} else if len(nameTag) > 2 {
		return fmt.Errorf("invalid image tag format: %s", SanitizeLogMessage(imageName))
	}

	return nil
}

// SecureReadFile reads a file with size limits and validation
func SecureReadFile(filename string) ([]byte, error) {
	// Use only the base filename to prevent path traversal
	baseName := filepath.Base(filename)
	if strings.Contains(baseName, "..") || baseName != filename {
		return nil, fmt.Errorf("path contains traversal sequences")
	}

	info, err := os.Stat(baseName)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %s", SanitizeErrorMessage(err))
	}

	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), MaxFileSize)
	}

	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", SanitizeLogMessage(baseName))
	}

	file, err := os.Open(baseName)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %s", SanitizeErrorMessage(err))
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, MaxFileSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %s", SanitizeErrorMessage(err))
	}

	if len(data) > MaxFileSize {
		return nil, fmt.Errorf("file size exceeds limit during read")
	}

	return data, nil
}

// SanitizeErrorMessage removes sensitive information from error messages
func SanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	errMsg = pathRegex.ReplaceAllString(errMsg, "[PATH_REDACTED]")
	errMsg = sensitiveDataRegex.ReplaceAllString(errMsg, "$1=[REDACTED]")
	errMsg = ipRegex.ReplaceAllString(errMsg, "[IP_REDACTED]")

	return errMsg
}

// SanitizeLogMessage removes sensitive information from log messages
func SanitizeLogMessage(message string) string {
	message = pathRegex.ReplaceAllString(message, "[PATH_REDACTED]")
	message = sensitiveDataRegex.ReplaceAllString(message, "$1=[REDACTED]")
	message = ipRegex.ReplaceAllString(message, "[IP_REDACTED]")
	return message
}

// CalculateImageHash calculates a hash of the pulled image for integrity verification
func CalculateImageHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
