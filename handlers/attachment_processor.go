package handlers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ccagent/core/log"
	"ccagent/models"
)

const (
	// MaxAttachmentSize is the maximum size for a single attachment (10MB)
	MaxAttachmentSize = 10 * 1024 * 1024
	// MaxTotalAttachmentsSize is the maximum total size for all attachments (50MB)
	MaxTotalAttachmentsSize = 50 * 1024 * 1024
)

// AttachmentProcessor handles processing of attachments from messages
type AttachmentProcessor struct {
	baseDir string
}

// NewAttachmentProcessor creates a new attachment processor
func NewAttachmentProcessor() *AttachmentProcessor {
	return &AttachmentProcessor{
		baseDir: "/tmp/ccagent-attachments",
	}
}

// ProcessAttachments decodes base64 attachments and saves them to temporary files
// Returns array of absolute file paths
func (ap *AttachmentProcessor) ProcessAttachments(attachments []models.Attachment, sessionID string) ([]string, error) {
	if len(attachments) == 0 {
		return nil, nil
	}

	log.Info("📋 Starting to process %d attachments for session %s", len(attachments), sessionID)

	// Create session-specific directory
	sessionDir := filepath.Join(ap.baseDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create attachments directory: %w", err)
	}

	var filePaths []string
	var totalSize int64

	for i, attachment := range attachments {
		// Validate attachment type
		if attachment.AttachmentType != "image" && attachment.AttachmentType != "other" {
			return nil, fmt.Errorf("invalid attachment type: %s (must be 'image' or 'other')", attachment.AttachmentType)
		}

		// Decode base64 content
		decodedContent, err := base64.StdEncoding.DecodeString(attachment.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode attachment %d: %w", i, err)
		}

		// Validate individual attachment size
		if len(decodedContent) > MaxAttachmentSize {
			return nil, fmt.Errorf("attachment %d exceeds maximum size of %d bytes", i, MaxAttachmentSize)
		}

		// Track total size
		totalSize += int64(len(decodedContent))
		if totalSize > MaxTotalAttachmentsSize {
			return nil, fmt.Errorf("total attachments size exceeds maximum of %d bytes", MaxTotalAttachmentsSize)
		}

		// Detect file extension
		extension := ap.detectFileExtension(attachment.AttachmentType, decodedContent)

		// Generate unique filename
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%d_attachment%s", timestamp, i, extension)
		filePath := filepath.Join(sessionDir, filename)

		// Write file with restrictive permissions
		if err := os.WriteFile(filePath, decodedContent, 0600); err != nil {
			return nil, fmt.Errorf("failed to write attachment %d to file: %w", i, err)
		}

		filePaths = append(filePaths, filePath)
		log.Info("✅ Saved attachment %d to %s (%d bytes)", i, filePath, len(decodedContent))
	}

	log.Info("📋 Successfully processed %d attachments, total size: %d bytes", len(attachments), totalSize)
	return filePaths, nil
}

// detectFileExtension determines the appropriate file extension based on type and content
func (ap *AttachmentProcessor) detectFileExtension(attachmentType string, content []byte) string {
	if attachmentType == "image" {
		return ap.detectImageExtension(content)
	}

	// For "other" type, try to detect common formats
	return ap.detectGenericExtension(content)
}

// detectImageExtension detects image format from magic bytes
func (ap *AttachmentProcessor) detectImageExtension(content []byte) string {
	if len(content) < 4 {
		return ".bin"
	}

	// PNG: 89 50 4E 47
	if bytes.HasPrefix(content, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return ".png"
	}

	// JPEG: FF D8 FF
	if bytes.HasPrefix(content, []byte{0xFF, 0xD8, 0xFF}) {
		return ".jpg"
	}

	// GIF: 47 49 46 38
	if bytes.HasPrefix(content, []byte{0x47, 0x49, 0x46, 0x38}) {
		return ".gif"
	}

	// WebP: RIFF....WEBP
	if len(content) >= 12 && bytes.HasPrefix(content, []byte{0x52, 0x49, 0x46, 0x46}) {
		if bytes.Equal(content[8:12], []byte{0x57, 0x45, 0x42, 0x50}) {
			return ".webp"
		}
	}

	// BMP: 42 4D
	if bytes.HasPrefix(content, []byte{0x42, 0x4D}) {
		return ".bmp"
	}

	// Default for unknown image formats
	return ".img"
}

// detectGenericExtension detects common file formats from magic bytes
func (ap *AttachmentProcessor) detectGenericExtension(content []byte) string {
	if len(content) < 4 {
		return ".bin"
	}

	// PDF: %PDF
	if bytes.HasPrefix(content, []byte{0x25, 0x50, 0x44, 0x46}) {
		return ".pdf"
	}

	// ZIP: PK (50 4B)
	if bytes.HasPrefix(content, []byte{0x50, 0x4B}) {
		return ".zip"
	}

	// Check for text content (common text file patterns)
	if ap.isLikelyText(content) {
		return ".txt"
	}

	// Default for unknown formats
	return ".bin"
}

// isLikelyText checks if content is likely text-based
func (ap *AttachmentProcessor) isLikelyText(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Sample first 512 bytes or entire content if smaller
	sampleSize := 512
	if len(content) < sampleSize {
		sampleSize = len(content)
	}

	// Count printable characters
	printableCount := 0
	for i := 0; i < sampleSize; i++ {
		b := content[i]
		// Check for printable ASCII (32-126) or common whitespace (9, 10, 13)
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 {
			printableCount++
		}
	}

	// If more than 90% of characters are printable, consider it text
	return float64(printableCount)/float64(sampleSize) > 0.9
}

// FormatAttachmentsForPrompt creates formatted text to append to user message
func (ap *AttachmentProcessor) FormatAttachmentsForPrompt(filePaths []string) string {
	if len(filePaths) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("\n\n---\nAttachments:\n")

	for _, path := range filePaths {
		builder.WriteString("- ")
		builder.WriteString(path)
		builder.WriteString("\n")
	}

	return builder.String()
}
