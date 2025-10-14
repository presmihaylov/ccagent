package utils

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccagent/models"
)

// Test magic bytes detection for various file types

func TestDetermineFileExtensionFromMagicBytes_PNG(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	ext := DetermineFileExtensionFromMagicBytes(pngHeader)
	if ext != ".png" {
		t.Errorf("Expected .png, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_JPEG(t *testing.T) {
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	ext := DetermineFileExtensionFromMagicBytes(jpegHeader)
	if ext != ".jpg" {
		t.Errorf("Expected .jpg, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_GIF87a(t *testing.T) {
	gifHeader := []byte("GIF87a")
	ext := DetermineFileExtensionFromMagicBytes(gifHeader)
	if ext != ".gif" {
		t.Errorf("Expected .gif, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_GIF89a(t *testing.T) {
	gifHeader := []byte("GIF89a")
	ext := DetermineFileExtensionFromMagicBytes(gifHeader)
	if ext != ".gif" {
		t.Errorf("Expected .gif, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_PDF(t *testing.T) {
	pdfHeader := []byte("%PDF-1.4\n")
	ext := DetermineFileExtensionFromMagicBytes(pdfHeader)
	if ext != ".pdf" {
		t.Errorf("Expected .pdf, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_ZIP(t *testing.T) {
	zipHeader := []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00}
	ext := DetermineFileExtensionFromMagicBytes(zipHeader)
	if ext != ".zip" {
		t.Errorf("Expected .zip, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_BMP(t *testing.T) {
	bmpHeader := []byte{0x42, 0x4D, 0x00, 0x00}
	ext := DetermineFileExtensionFromMagicBytes(bmpHeader)
	if ext != ".bmp" {
		t.Errorf("Expected .bmp, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_ICO(t *testing.T) {
	icoHeader := []byte{0x00, 0x00, 0x01, 0x00}
	ext := DetermineFileExtensionFromMagicBytes(icoHeader)
	if ext != ".ico" {
		t.Errorf("Expected .ico, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_GZIP(t *testing.T) {
	gzipHeader := []byte{0x1F, 0x8B, 0x08, 0x00}
	ext := DetermineFileExtensionFromMagicBytes(gzipHeader)
	if ext != ".gz" {
		t.Errorf("Expected .gz, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_RAR(t *testing.T) {
	rarHeader := []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07}
	ext := DetermineFileExtensionFromMagicBytes(rarHeader)
	if ext != ".rar" {
		t.Errorf("Expected .rar, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_7Z(t *testing.T) {
	sevenZHeader := []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}
	ext := DetermineFileExtensionFromMagicBytes(sevenZHeader)
	if ext != ".7z" {
		t.Errorf("Expected .7z, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_BZ2(t *testing.T) {
	bz2Header := []byte{0x42, 0x5A, 0x68, 0x39}
	ext := DetermineFileExtensionFromMagicBytes(bz2Header)
	if ext != ".bz2" {
		t.Errorf("Expected .bz2, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_WebP(t *testing.T) {
	webpHeader := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}
	ext := DetermineFileExtensionFromMagicBytes(webpHeader)
	if ext != ".webp" {
		t.Errorf("Expected .webp, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_OldMSOffice(t *testing.T) {
	docHeader := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1}
	ext := DetermineFileExtensionFromMagicBytes(docHeader)
	if ext != ".doc" {
		t.Errorf("Expected .doc, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_ShellScript(t *testing.T) {
	shellHeader := []byte("#!/bin/bash\n")
	ext := DetermineFileExtensionFromMagicBytes(shellHeader)
	if ext != ".sh" {
		t.Errorf("Expected .sh, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_PHP(t *testing.T) {
	phpHeader := []byte("<?php\necho 'test';")
	ext := DetermineFileExtensionFromMagicBytes(phpHeader)
	if ext != ".php" {
		t.Errorf("Expected .php, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_XML(t *testing.T) {
	xmlHeader := []byte("<?xml version=\"1.0\"?>")
	ext := DetermineFileExtensionFromMagicBytes(xmlHeader)
	if ext != ".xml" {
		t.Errorf("Expected .xml, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_PlainText(t *testing.T) {
	textContent := []byte("This is a plain text file with some content.\nIt has multiple lines.\n")
	ext := DetermineFileExtensionFromMagicBytes(textContent)
	if ext != ".txt" {
		t.Errorf("Expected .txt, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_Unknown(t *testing.T) {
	unknownHeader := []byte{0x00, 0xFF, 0xAA, 0xBB, 0xCC, 0xDD}
	ext := DetermineFileExtensionFromMagicBytes(unknownHeader)
	if ext != ".bin" {
		t.Errorf("Expected .bin, got %s", ext)
	}
}

func TestDetermineFileExtensionFromMagicBytes_Empty(t *testing.T) {
	emptyContent := []byte{}
	ext := DetermineFileExtensionFromMagicBytes(emptyContent)
	if ext != ".bin" {
		t.Errorf("Expected .bin for empty content, got %s", ext)
	}
}

// Test attachment storage functions

func TestDecodeAndStoreAttachment_ValidPNG(t *testing.T) {
	// Create a PNG header as base64
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	base64Content := base64.StdEncoding.EncodeToString(pngHeader)

	attachment := models.Attachment{
		Content:        base64Content,
		AttachmentType: models.AttachmentTypeImage,
	}

	sessionID := "test_session_png"
	filePath, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s", filePath)
	}

	// Verify file has .png extension
	if !strings.HasSuffix(filePath, ".png") {
		t.Errorf("Expected .png extension, got: %s", filePath)
	}

	// Verify file path contains session ID
	if !strings.Contains(filePath, sessionID) {
		t.Errorf("Expected file path to contain session ID %s, got: %s", sessionID, filePath)
	}

	// Verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if len(content) != len(pngHeader) {
		t.Errorf("Expected content length %d, got %d", len(pngHeader), len(content))
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_ValidJPEG(t *testing.T) {
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	base64Content := base64.StdEncoding.EncodeToString(jpegHeader)

	attachment := models.Attachment{
		Content:        base64Content,
		AttachmentType: models.AttachmentTypeImage,
	}

	sessionID := "test_session_jpeg"
	filePath, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(filePath, ".jpg") {
		t.Errorf("Expected .jpg extension, got: %s", filePath)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_ValidPDF(t *testing.T) {
	pdfHeader := []byte("%PDF-1.4\n")
	base64Content := base64.StdEncoding.EncodeToString(pdfHeader)

	attachment := models.Attachment{
		Content:        base64Content,
		AttachmentType: models.AttachmentTypeOther,
	}

	sessionID := "test_session_pdf"
	filePath, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.HasSuffix(filePath, ".pdf") {
		t.Errorf("Expected .pdf extension, got: %s", filePath)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_InvalidBase64(t *testing.T) {
	attachment := models.Attachment{
		Content:        "not-valid-base64!!!",
		AttachmentType: models.AttachmentTypeOther,
	}

	sessionID := "test_session_invalid"
	_, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}

	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("Expected 'invalid base64' error, got: %v", err)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_EmptyContent(t *testing.T) {
	attachment := models.Attachment{
		Content:        "",
		AttachmentType: models.AttachmentTypeOther,
	}

	sessionID := "test_session_empty"
	_, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err == nil {
		t.Error("Expected error for empty content, got nil")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected 'empty' error, got: %v", err)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_EmptyDecodedContent(t *testing.T) {
	// Base64 encoding of empty string
	emptyBase64 := base64.StdEncoding.EncodeToString([]byte{})

	attachment := models.Attachment{
		Content:        emptyBase64,
		AttachmentType: models.AttachmentTypeOther,
	}

	sessionID := "test_session_decoded_empty"
	_, err := DecodeAndStoreAttachment(attachment, sessionID, 0)

	if err == nil {
		t.Error("Expected error for empty decoded content, got nil")
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestDecodeAndStoreAttachment_MultipleAttachments(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	pngBase64 := base64.StdEncoding.EncodeToString(pngHeader)
	jpegBase64 := base64.StdEncoding.EncodeToString(jpegHeader)

	attachments := []models.Attachment{
		{Content: pngBase64, AttachmentType: models.AttachmentTypeImage},
		{Content: jpegBase64, AttachmentType: models.AttachmentTypeImage},
	}

	sessionID := "test_session_multiple"
	var paths []string

	for i, att := range attachments {
		filePath, err := DecodeAndStoreAttachment(att, sessionID, i)
		if err != nil {
			t.Fatalf("Failed to store attachment %d: %v", i, err)
		}
		paths = append(paths, filePath)
	}

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	if !strings.Contains(paths[0], "attachment_0") {
		t.Errorf("Expected first file to contain 'attachment_0', got: %s", paths[0])
	}

	if !strings.Contains(paths[1], "attachment_1") {
		t.Errorf("Expected second file to contain 'attachment_1', got: %s", paths[1])
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

// Test directory functions

func TestGetAttachmentsDir_CreatesPath(t *testing.T) {
	sessionID := "test_session_dir"
	dir, err := GetAttachmentsDir(sessionID)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedPath := filepath.Join("/tmp", "ccagent", "attachments", sessionID)
	if dir != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, dir)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

func TestGetAttachmentsDir_CreatesDirectory(t *testing.T) {
	sessionID := "test_session_create_dir"
	dir, err := GetAttachmentsDir(sessionID)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("Expected directory to exist at %s", dir)
	}

	if !info.IsDir() {
		t.Errorf("Expected %s to be a directory", dir)
	}

	// Cleanup
	os.RemoveAll(filepath.Join("/tmp", "ccagent", "attachments", sessionID))
}

// Test formatting functions

func TestFormatAttachmentsText_SingleFile(t *testing.T) {
	paths := []string{"/tmp/ccagent/attachments/sess1/attachment_0.png"}
	text := FormatAttachmentsText(paths)

	expectedSubstrings := []string{
		"---",
		"Attachments:",
		"/tmp/ccagent/attachments/sess1/attachment_0.png",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(text, substr) {
			t.Errorf("Expected text to contain '%s', got: %s", substr, text)
		}
	}
}

func TestFormatAttachmentsText_MultipleFiles(t *testing.T) {
	paths := []string{
		"/tmp/ccagent/attachments/sess1/attachment_0.png",
		"/tmp/ccagent/attachments/sess1/attachment_1.pdf",
		"/tmp/ccagent/attachments/sess1/attachment_2.txt",
	}
	text := FormatAttachmentsText(paths)

	expectedSubstrings := []string{
		"---",
		"Attachments:",
		"/tmp/ccagent/attachments/sess1/attachment_0.png",
		"/tmp/ccagent/attachments/sess1/attachment_1.pdf",
		"/tmp/ccagent/attachments/sess1/attachment_2.txt",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(text, substr) {
			t.Errorf("Expected text to contain '%s', got: %s", substr, text)
		}
	}

	// Verify each path is on its own line with a bullet
	for _, path := range paths {
		expected := "- " + path
		if !strings.Contains(text, expected) {
			t.Errorf("Expected text to contain '%s'", expected)
		}
	}
}

func TestFormatAttachmentsText_EmptyList(t *testing.T) {
	paths := []string{}
	text := FormatAttachmentsText(paths)

	if text != "" {
		t.Errorf("Expected empty string for empty paths, got: %s", text)
	}
}

func TestFormatAttachmentsText_NilList(t *testing.T) {
	var paths []string
	text := FormatAttachmentsText(paths)

	if text != "" {
		t.Errorf("Expected empty string for nil paths, got: %s", text)
	}
}
