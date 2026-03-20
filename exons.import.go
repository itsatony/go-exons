package exons

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
)

// ImportResult holds the result of a document import operation.
// Spec is the parsed document, Resources holds any additional files
// found in a zip archive alongside the main document.
type ImportResult struct {
	Spec      *Spec
	Resources map[string][]byte
}

// Import parses a document from raw bytes and a filename hint.
// The filename extension determines the format:
//   - ".md" → parse as markdown with YAML frontmatter
//   - ".zip" → parse as a zip archive containing a document + resources
//   - unknown/empty → try as markdown
//
// Returns an ImportResult with the parsed Spec and any resources (zip only).
func Import(data []byte, filename string) (*ImportResult, error) {
	if len(data) == 0 {
		return nil, NewImportError(ErrMsgImportFailed, nil)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case FileExtensionZip:
		return ImportDirectory(data)
	case FileExtensionMarkdown:
		return importMarkdown(data)
	default:
		// Unknown extension — try as markdown
		return importMarkdown(data)
	}
}

// importMarkdown parses a markdown document with YAML frontmatter.
func importMarkdown(data []byte) (*ImportResult, error) {
	spec, err := Parse(data)
	if err != nil {
		return nil, NewImportError(ErrMsgImportReadFailed, err)
	}
	return &ImportResult{
		Spec:      spec,
		Resources: make(map[string][]byte),
	}, nil
}

// ImportDirectory imports a document from a zip archive.
// The archive must contain exactly one of: SKILL.md, AGENT.md, or PROMPT.md
// (case-insensitive base name match). If multiple document files are found,
// an error is returned. All other non-directory files are collected as resources.
// Individual resource files are limited to MaxImportResourceSize bytes.
func ImportDirectory(data []byte) (*ImportResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, NewImportError(ErrMsgImportZipFailed, err)
	}

	// Pre-compute upper-case document filenames for comparison
	skillUpper := strings.ToUpper(DocumentFilenameSkill)
	agentUpper := strings.ToUpper(DocumentFilenameAgent)
	promptUpper := strings.ToUpper(DocumentFilenamePrompt)

	var documentData []byte
	resources := make(map[string][]byte)

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		baseName := strings.ToUpper(filepath.Base(file.Name))
		if baseName == skillUpper || baseName == agentUpper || baseName == promptUpper {
			// Detect multiple document files
			if documentData != nil {
				return nil, NewImportError(ErrMsgImportMultipleDocuments, nil)
			}
			docBytes, readErr := readZipEntry(file, 0)
			if readErr != nil {
				return nil, NewImportError(ErrMsgImportReadFailed, readErr)
			}
			documentData = docBytes
		} else {
			resBytes, readErr := readZipEntry(file, MaxImportResourceSize)
			if readErr != nil {
				return nil, NewImportError(ErrMsgImportReadFailed, readErr)
			}
			resources[file.Name] = resBytes
		}
	}

	if documentData == nil {
		return nil, NewImportError(ErrMsgImportNoDocument, nil)
	}

	spec, err := Parse(documentData)
	if err != nil {
		return nil, NewImportError(ErrMsgImportReadFailed, err)
	}

	return &ImportResult{
		Spec:      spec,
		Resources: resources,
	}, nil
}

// readZipEntry reads a zip file entry with an optional size limit.
// If maxSize is 0 or negative, no limit is applied (used for document files
// which are size-checked at parse time via DefaultMaxFrontmatterSize).
func readZipEntry(file *zip.File, maxSize int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var reader io.Reader = rc
	if maxSize > 0 {
		reader = io.LimitReader(rc, maxSize+1) // +1 to detect overflow
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if maxSize > 0 && int64(len(data)) > maxSize {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}
