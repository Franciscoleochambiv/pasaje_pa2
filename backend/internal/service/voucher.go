package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VoucherService handles voucher image storage and validation.
type VoucherService struct {
	StoragePath string
	MaxSizeMB   int
}

// ValidateImage checks that the uploaded file is a valid image within size limits.
func (s *VoucherService) ValidateImage(header *multipart.FileHeader) error {
	maxBytes := int64(s.MaxSizeMB) * 1024 * 1024
	if header.Size > maxBytes {
		return fmt.Errorf("el archivo excede el limite de %dMB", s.MaxSizeMB)
	}

	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo: %w", err)
	}
	defer file.Close()

	// Read first 12 bytes to check magic bytes
	magic := make([]byte, 12)
	n, err := file.Read(magic)
	if err != nil || n < 4 {
		return fmt.Errorf("archivo no valido")
	}

	if !isJPEG(magic) && !isPNG(magic) && !isWebP(magic[:n]) {
		return fmt.Errorf("formato no soportado (solo JPEG, PNG, WebP)")
	}

	return nil
}

// SaveVoucher saves the uploaded file to the storage path and returns the relative path.
func (s *VoucherService) SaveVoucher(file io.Reader, reservationCode string, ext string) (string, error) {
	now := time.Now()
	subDir := filepath.Join(fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	fullDir := filepath.Join(s.StoragePath, subDir)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", fmt.Errorf("crear directorio: %w", err)
	}

	filename := fmt.Sprintf("%s_%d%s", reservationCode, now.UnixMilli(), ext)
	relPath := filepath.Join(subDir, filename)
	fullPath := filepath.Join(s.StoragePath, relPath)

	out, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("crear archivo: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("escribir archivo: %w", err)
	}

	return relPath, nil
}

// GetFullPath returns the absolute path for a relative voucher path.
func (s *VoucherService) GetFullPath(relativePath string) string {
	return filepath.Join(s.StoragePath, relativePath)
}

// ExtFromHeader extracts the file extension from a multipart file header.
func ExtFromHeader(header *multipart.FileHeader) string {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	return ext
}

func isJPEG(b []byte) bool {
	return len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF
}

func isPNG(b []byte) bool {
	return len(b) >= 4 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47
}

func isWebP(b []byte) bool {
	return len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP"
}
