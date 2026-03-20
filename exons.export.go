package exons

import (
	"archive/zip"
	"bytes"
)

// ExportDirectory exports a Spec and optional resources as a zip archive.
// The spec is serialized using ExportFull() and placed in the archive
// with the appropriate document filename (SKILL.md, AGENT.md, or PROMPT.md).
// Additional resources are included as-is in the archive.
func ExportDirectory(spec *Spec, resources map[string][]byte) ([]byte, error) {
	if spec == nil {
		return nil, NewExportError(ErrMsgExportFailed, nil)
	}

	docBytes, err := spec.ExportFull()
	if err != nil {
		return nil, NewExportError(ErrMsgExportZipFailed, err)
	}

	filename := documentFilename(spec.EffectiveType())

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Write the main document
	f, err := w.Create(filename)
	if err != nil {
		return nil, NewExportError(ErrMsgExportZipFailed, err)
	}
	if _, err := f.Write(docBytes); err != nil {
		return nil, NewExportError(ErrMsgExportZipFailed, err)
	}

	// Write resources
	for name, data := range resources {
		rf, resErr := w.Create(name)
		if resErr != nil {
			return nil, NewExportError(ErrMsgExportZipFailed, resErr)
		}
		if _, resErr := rf.Write(data); resErr != nil {
			return nil, NewExportError(ErrMsgExportZipFailed, resErr)
		}
	}

	if err := w.Close(); err != nil {
		return nil, NewExportError(ErrMsgExportZipFailed, err)
	}

	return buf.Bytes(), nil
}

// documentFilename returns the appropriate document filename for the given type.
func documentFilename(dt DocumentType) string {
	switch dt {
	case DocumentTypeAgent:
		return DocumentFilenameAgent
	case DocumentTypePrompt:
		return DocumentFilenamePrompt
	default:
		return DocumentFilenameSkill
	}
}
