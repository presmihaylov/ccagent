package handlers

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccagent/models"
)

func TestProcessAttachments_EmptyAttachments(t *testing.T) {
	processor := NewAttachmentProcessor()
	filePaths, err := processor.ProcessAttachments([]models.Attachment{}, "test-session")

	if err != nil {
		t.Errorf("Expected no error for empty attachments, got: %v", err)
	}

	if filePaths != nil {
		t.Errorf("Expected nil for empty attachments, got: %v", filePaths)
	}
}

func TestProcessAttachments_ValidImagePNG(t *testing.T) {
	processor := NewAttachmentProcessor()

	// PNG magic bytes
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	encodedData := base64.StdEncoding.EncodeToString(pngData)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "image",
		},
	}

	filePaths, err := processor.ProcessAttachments(attachments, "test-session-png")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(filePaths) != 1 {
		t.Fatalf("Expected 1 file path, got %d", len(filePaths))
	}

	// Verify file exists and has correct extension
	if !strings.HasSuffix(filePaths[0], ".png") {
		t.Errorf("Expected .png extension, got: %s", filePaths[0])
	}

	// Verify file content
	content, err := os.ReadFile(filePaths[0])
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != string(pngData) {
		t.Errorf("File content doesn't match original data")
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(filePaths[0]))
}

func TestProcessAttachments_ValidImageJPEG(t *testing.T) {
	processor := NewAttachmentProcessor()

	// JPEG magic bytes
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	encodedData := base64.StdEncoding.EncodeToString(jpegData)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "image",
		},
	}

	filePaths, err := processor.ProcessAttachments(attachments, "test-session-jpeg")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(filePaths[0], ".jpg") {
		t.Errorf("Expected .jpg extension, got: %s", filePaths[0])
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(filePaths[0]))
}

func TestProcessAttachments_ValidPDF(t *testing.T) {
	processor := NewAttachmentProcessor()

	// PDF magic bytes
	pdfData := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	encodedData := base64.StdEncoding.EncodeToString(pdfData)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "other",
		},
	}

	filePaths, err := processor.ProcessAttachments(attachments, "test-session-pdf")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(filePaths[0], ".pdf") {
		t.Errorf("Expected .pdf extension, got: %s", filePaths[0])
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(filePaths[0]))
}

func TestProcessAttachments_MultipleAttachments(t *testing.T) {
	processor := NewAttachmentProcessor()

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pdfData := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}

	attachments := []models.Attachment{
		{
			Content:        base64.StdEncoding.EncodeToString(pngData),
			AttachmentType: "image",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(pdfData),
			AttachmentType: "other",
		},
	}

	filePaths, err := processor.ProcessAttachments(attachments, "test-session-multi")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(filePaths) != 2 {
		t.Fatalf("Expected 2 file paths, got %d", len(filePaths))
	}

	if !strings.HasSuffix(filePaths[0], ".png") {
		t.Errorf("Expected first file to have .png extension, got: %s", filePaths[0])
	}

	if !strings.HasSuffix(filePaths[1], ".pdf") {
		t.Errorf("Expected second file to have .pdf extension, got: %s", filePaths[1])
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(filePaths[0]))
}

func TestProcessAttachments_InvalidBase64(t *testing.T) {
	processor := NewAttachmentProcessor()

	attachments := []models.Attachment{
		{
			Content:        "not-valid-base64!@#$%",
			AttachmentType: "image",
		},
	}

	_, err := processor.ProcessAttachments(attachments, "test-session-invalid")
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode") {
		t.Errorf("Expected 'failed to decode' in error message, got: %v", err)
	}
}

func TestProcessAttachments_InvalidAttachmentType(t *testing.T) {
	processor := NewAttachmentProcessor()

	data := []byte("test data")
	encodedData := base64.StdEncoding.EncodeToString(data)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "invalid-type",
		},
	}

	_, err := processor.ProcessAttachments(attachments, "test-session-invalid-type")
	if err == nil {
		t.Error("Expected error for invalid attachment type, got nil")
	}

	if !strings.Contains(err.Error(), "invalid attachment type") {
		t.Errorf("Expected 'invalid attachment type' in error message, got: %v", err)
	}
}

func TestProcessAttachments_AttachmentTooLarge(t *testing.T) {
	processor := NewAttachmentProcessor()

	// Create data larger than MaxAttachmentSize (10MB)
	largeData := make([]byte, MaxAttachmentSize+1)
	encodedData := base64.StdEncoding.EncodeToString(largeData)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "other",
		},
	}

	_, err := processor.ProcessAttachments(attachments, "test-session-large")
	if err == nil {
		t.Error("Expected error for oversized attachment, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("Expected 'exceeds maximum size' in error message, got: %v", err)
	}
}

func TestProcessAttachments_TotalSizeTooLarge(t *testing.T) {
	processor := NewAttachmentProcessor()

	// Create multiple attachments that total more than MaxTotalAttachmentsSize (50MB)
	// Each attachment is 9MB (under individual limit), total is 54MB (over total limit)
	data := make([]byte, 9*1024*1024)

	attachments := []models.Attachment{
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
		{
			Content:        base64.StdEncoding.EncodeToString(data),
			AttachmentType: "other",
		},
	}

	_, err := processor.ProcessAttachments(attachments, "test-session-total-large")
	if err == nil {
		t.Error("Expected error for total size exceeding limit, got nil")
	}

	if !strings.Contains(err.Error(), "total attachments size exceeds") {
		t.Errorf("Expected 'total attachments size exceeds' in error message, got: %v", err)
	}
}

func TestDetectImageExtension_PNG(t *testing.T) {
	processor := NewAttachmentProcessor()
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	ext := processor.detectImageExtension(pngData)
	if ext != ".png" {
		t.Errorf("Expected .png, got: %s", ext)
	}
}

func TestDetectImageExtension_JPEG(t *testing.T) {
	processor := NewAttachmentProcessor()
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	ext := processor.detectImageExtension(jpegData)
	if ext != ".jpg" {
		t.Errorf("Expected .jpg, got: %s", ext)
	}
}

func TestDetectImageExtension_GIF(t *testing.T) {
	processor := NewAttachmentProcessor()
	gifData := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}

	ext := processor.detectImageExtension(gifData)
	if ext != ".gif" {
		t.Errorf("Expected .gif, got: %s", ext)
	}
}

func TestDetectImageExtension_BMP(t *testing.T) {
	processor := NewAttachmentProcessor()
	bmpData := []byte{0x42, 0x4D, 0x00, 0x00}

	ext := processor.detectImageExtension(bmpData)
	if ext != ".bmp" {
		t.Errorf("Expected .bmp, got: %s", ext)
	}
}

func TestDetectImageExtension_WebP(t *testing.T) {
	processor := NewAttachmentProcessor()
	webpData := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}

	ext := processor.detectImageExtension(webpData)
	if ext != ".webp" {
		t.Errorf("Expected .webp, got: %s", ext)
	}
}

func TestDetectImageExtension_Unknown(t *testing.T) {
	processor := NewAttachmentProcessor()
	unknownData := []byte{0x00, 0x00, 0x00, 0x00}

	ext := processor.detectImageExtension(unknownData)
	if ext != ".img" {
		t.Errorf("Expected .img for unknown format, got: %s", ext)
	}
}

func TestDetectGenericExtension_PDF(t *testing.T) {
	processor := NewAttachmentProcessor()
	pdfData := []byte{0x25, 0x50, 0x44, 0x46, 0x2D}

	ext := processor.detectGenericExtension(pdfData)
	if ext != ".pdf" {
		t.Errorf("Expected .pdf, got: %s", ext)
	}
}

func TestDetectGenericExtension_ZIP(t *testing.T) {
	processor := NewAttachmentProcessor()
	zipData := []byte{0x50, 0x4B, 0x03, 0x04}

	ext := processor.detectGenericExtension(zipData)
	if ext != ".zip" {
		t.Errorf("Expected .zip, got: %s", ext)
	}
}

func TestDetectGenericExtension_Text(t *testing.T) {
	processor := NewAttachmentProcessor()
	textData := []byte("This is a plain text file with normal characters")

	ext := processor.detectGenericExtension(textData)
	if ext != ".txt" {
		t.Errorf("Expected .txt, got: %s", ext)
	}
}

func TestDetectGenericExtension_Binary(t *testing.T) {
	processor := NewAttachmentProcessor()
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}

	ext := processor.detectGenericExtension(binaryData)
	if ext != ".bin" {
		t.Errorf("Expected .bin for unknown binary, got: %s", ext)
	}
}

func TestIsLikelyText_PlainText(t *testing.T) {
	processor := NewAttachmentProcessor()
	textData := []byte("This is plain text with letters, numbers 123, and punctuation!")

	if !processor.isLikelyText(textData) {
		t.Error("Expected plain text to be detected as text")
	}
}

func TestIsLikelyText_Binary(t *testing.T) {
	processor := NewAttachmentProcessor()
	binaryData := make([]byte, 512)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}

	if processor.isLikelyText(binaryData) {
		t.Error("Expected binary data to not be detected as text")
	}
}

func TestIsLikelyText_Empty(t *testing.T) {
	processor := NewAttachmentProcessor()
	emptyData := []byte{}

	if processor.isLikelyText(emptyData) {
		t.Error("Expected empty data to not be detected as text")
	}
}

func TestFormatAttachmentsForPrompt_Empty(t *testing.T) {
	processor := NewAttachmentProcessor()
	result := processor.FormatAttachmentsForPrompt([]string{})

	if result != "" {
		t.Errorf("Expected empty string for no attachments, got: %s", result)
	}
}

func TestFormatAttachmentsForPrompt_SingleFile(t *testing.T) {
	processor := NewAttachmentProcessor()
	filePaths := []string{"/tmp/test/file1.png"}

	result := processor.FormatAttachmentsForPrompt(filePaths)

	expected := "\n\n---\nAttachments:\n- /tmp/test/file1.png\n"
	if result != expected {
		t.Errorf("Expected:\n%s\n\nGot:\n%s", expected, result)
	}
}

func TestFormatAttachmentsForPrompt_MultipleFiles(t *testing.T) {
	processor := NewAttachmentProcessor()
	filePaths := []string{
		"/tmp/test/file1.png",
		"/tmp/test/file2.pdf",
		"/tmp/test/file3.txt",
	}

	result := processor.FormatAttachmentsForPrompt(filePaths)

	if !strings.Contains(result, "/tmp/test/file1.png") {
		t.Error("Expected result to contain file1.png")
	}
	if !strings.Contains(result, "/tmp/test/file2.pdf") {
		t.Error("Expected result to contain file2.pdf")
	}
	if !strings.Contains(result, "/tmp/test/file3.txt") {
		t.Error("Expected result to contain file3.txt")
	}
	if !strings.Contains(result, "Attachments:") {
		t.Error("Expected result to contain 'Attachments:' header")
	}
}

func TestProcessAttachments_CreatesSessionDirectory(t *testing.T) {
	processor := NewAttachmentProcessor()

	data := []byte("test data")
	encodedData := base64.StdEncoding.EncodeToString(data)

	attachments := []models.Attachment{
		{
			Content:        encodedData,
			AttachmentType: "other",
		},
	}

	sessionID := "test-session-dir-creation"
	filePaths, err := processor.ProcessAttachments(attachments, sessionID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify session directory was created
	expectedDir := filepath.Join("/tmp/ccagent-attachments", sessionID)
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Expected session directory %s to exist", expectedDir)
	}

	// Verify file is in session directory
	if !strings.Contains(filePaths[0], sessionID) {
		t.Errorf("Expected file path to contain session ID, got: %s", filePaths[0])
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(filePaths[0]))
}
