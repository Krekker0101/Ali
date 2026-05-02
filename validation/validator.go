package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError represents a validation failure with field context
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// Error implements error interface
func (ve ValidationError) Error() string {
	if ve.Field != "" {
		return fmt.Sprintf("validation error in field '%s': %s", ve.Field, ve.Message)
	}
	return fmt.Sprintf("validation error: %s", ve.Message)
}

// Validator provides common validation utilities
type Validator struct {
	errors []ValidationError
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		errors: make([]ValidationError, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message, code string) *Validator {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
	return v
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() []ValidationError {
	return v.errors
}

// FirstError returns the first error or nil
func (v *Validator) FirstError() error {
	if len(v.errors) > 0 {
		return v.errors[0]
	}
	return nil
}

// String returns formatted error string
func (v *Validator) String() string {
	if !v.HasErrors() {
		return ""
	}
	var msgs []string
	for _, err := range v.errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// ValidateModelName validates a model name
func (v *Validator) ValidateModelName(name string) *Validator {
	if strings.TrimSpace(name) == "" {
		return v.AddError("modelName", "model name cannot be empty", "EMPTY")
	}
	
	if len(name) > 256 {
		return v.AddError("modelName", "model name cannot exceed 256 characters", "TOO_LONG")
	}
	
	// Allow alphanumeric, dots, slashes, colons, dashes, underscores
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._:/\-]+$`)
	if !pattern.MatchString(name) {
		return v.AddError("modelName", "model name contains invalid characters", "INVALID_CHARS")
	}
	
	return v
}

// ValidateFilePath validates a file path for security (prevents traversal attacks)
func (v *Validator) ValidateFilePath(path string, baseDir string) *Validator {
	if strings.TrimSpace(path) == "" {
		return v.AddError("filePath", "file path cannot be empty", "EMPTY")
	}
	
	// Prevent absolute paths
	if filepath.IsAbs(path) {
		return v.AddError("filePath", "absolute paths are not allowed", "ABSOLUTE_PATH")
	}
	
	// Resolve to absolute path and check if it's within baseDir
	absPath := filepath.Join(baseDir, path)
	absBase := filepath.Join(baseDir)
	
	// Normalize paths for comparison
	absPath = filepath.Clean(absPath)
	absBase = filepath.Clean(absBase)
	
	// Check for path traversal
	if !strings.HasPrefix(absPath, absBase) {
		return v.AddError("filePath", "path traversal detected", "TRAVERSAL")
	}
	
	return v
}

// ValidateFileSize validates a file size
func (v *Validator) ValidateFileSize(size int64, maxSizeBytes int64) *Validator {
	if size < 0 {
		return v.AddError("fileSize", "file size cannot be negative", "NEGATIVE")
	}
	
	if size > maxSizeBytes {
		return v.AddError("fileSize", 
			fmt.Sprintf("file size exceeds limit (%.2f MB)", float64(maxSizeBytes)/1024/1024), 
			"TOO_LARGE")
	}
	
	return v
}

// ValidateFileName validates a file name
func (v *Validator) ValidateFileName(name string) *Validator {
	if strings.TrimSpace(name) == "" {
		return v.AddError("fileName", "file name cannot be empty", "EMPTY")
	}
	
	if len(name) > 255 {
		return v.AddError("fileName", "file name exceeds 255 characters", "TOO_LONG")
	}
	
	// Check for directory separators
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return v.AddError("fileName", "file name cannot contain path separators", "PATH_SEPARATOR")
	}
	
	// Check for null bytes
	if strings.Contains(name, "\x00") {
		return v.AddError("fileName", "file name contains null bytes", "NULL_BYTE")
	}
	
	return v
}

// ValidatePort validates a port number
func (v *Validator) ValidatePort(port int) *Validator {
	if port < 1 || port > 65535 {
		return v.AddError("port", "port must be between 1 and 65535", "OUT_OF_RANGE")
	}
	return v
}

// ValidateTimeout validates a timeout in seconds
func (v *Validator) ValidateTimeout(timeoutSeconds int) *Validator {
	if timeoutSeconds < 1 {
		return v.AddError("timeout", "timeout must be at least 1 second", "TOO_SHORT")
	}
	
	if timeoutSeconds > 3600 {
		return v.AddError("timeout", "timeout cannot exceed 1 hour", "TOO_LONG")
	}
	
	return v
}

// ValidateFileExtension validates a file extension against a whitelist
func (v *Validator) ValidateFileExtension(filename string, allowedExtensions []string) *Validator {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return v.AddError("fileExtension", "file has no extension", "NO_EXTENSION")
	}
	
	// Remove leading dot for comparison
	ext = strings.TrimPrefix(ext, ".")
	
	found := false
	for _, allowed := range allowedExtensions {
		if strings.EqualFold(ext, allowed) {
			found = true
			break
		}
	}
	
	if !found {
		return v.AddError("fileExtension", 
			fmt.Sprintf("file extension '.%s' is not allowed", ext), 
			"NOT_ALLOWED")
	}
	
	return v
}

// ValidateNumCtx validates context size
func (v *Validator) ValidateNumCtx(numCtx int) *Validator {
	if numCtx < 4 {
		return v.AddError("numCtx", "context size must be at least 4", "TOO_SMALL")
	}
	
	if numCtx > 1000000 {
		return v.AddError("numCtx", "context size cannot exceed 1,000,000", "TOO_LARGE")
	}
	
	return v
}

// ValidateTemperature validates sampling temperature
func (v *Validator) ValidateTemperature(temperature float32) *Validator {
	if temperature < 0.0 {
		return v.AddError("temperature", "temperature cannot be negative", "NEGATIVE")
	}
	
	if temperature > 2.0 {
		return v.AddError("temperature", "temperature cannot exceed 2.0", "TOO_HIGH")
	}
	
	return v
}

// ValidateTopP validates top-p sampling parameter
func (v *Validator) ValidateTopP(topP float32) *Validator {
	if topP < 0.0 || topP > 1.0 {
		return v.AddError("topP", "top-p must be between 0.0 and 1.0", "OUT_OF_RANGE")
	}
	
	return v
}

// ValidateTopK validates top-k sampling parameter
func (v *Validator) ValidateTopK(topK int) *Validator {
	if topK < 0 {
		return v.AddError("topK", "top-k cannot be negative", "NEGATIVE")
	}
	
	if topK > 100000 {
		return v.AddError("topK", "top-k cannot exceed 100,000", "TOO_LARGE")
	}
	
	return v
}
