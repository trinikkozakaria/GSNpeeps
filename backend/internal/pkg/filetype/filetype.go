// Package filetype memvalidasi berkas unggahan berdasarkan extension, MIME type, dan file
// signature. Validasi dilakukan di backend karena client tidak pernah menjadi otoritas.
package filetype

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
)

var (
	// ErrUnsupported dikembalikan ketika extension, MIME, atau signature tidak termasuk
	// format yang disetujui kontrak.
	ErrUnsupported = errors.New("unsupported file type")
	// ErrSignatureMismatch dikembalikan ketika isi berkas tidak cocok dengan extension.
	ErrSignatureMismatch = errors.New("file signature does not match extension")
)

type family int

const (
	familyPDF family = iota
	familyJPEG
	familyPNG
	familyOLE2 // DOC, XLS, PPT lama
	familyOOXML
)

type descriptor struct {
	mediaType string
	family    family
}

// documentTypes adalah daftar format yang disetujui untuk dokumen karyawan:
// PDF, JPG, PNG, DOC, DOCX, XLS, XLSX, PPT, dan PPTX. Arsip seperti ZIP dan RAR tidak
// termasuk dan tetap ditolak meskipun DOCX secara fisik merupakan container ZIP.
var documentTypes = map[string]descriptor{
	".pdf":  {"application/pdf", familyPDF},
	".jpg":  {"image/jpeg", familyJPEG},
	".jpeg": {"image/jpeg", familyJPEG},
	".png":  {"image/png", familyPNG},
	".doc":  {"application/msword", familyOLE2},
	".xls":  {"application/vnd.ms-excel", familyOLE2},
	".ppt":  {"application/vnd.ms-powerpoint", familyOLE2},
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", familyOOXML},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", familyOOXML},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", familyOOXML},
}

// photoTypes membatasi foto absensi pada JPG dan PNG sesuai kontrak check-in.
var photoTypes = map[string]descriptor{
	".jpg":  {"image/jpeg", familyJPEG},
	".jpeg": {"image/jpeg", familyJPEG},
	".png":  {"image/png", familyPNG},
}

// supportingTypes membatasi dokumen pendukung pengajuan pada PDF, JPG, dan PNG sesuai
// kontrak ketidakhadiran dan lembur.
var supportingTypes = map[string]descriptor{
	".pdf":  {"application/pdf", familyPDF},
	".jpg":  {"image/jpeg", familyJPEG},
	".jpeg": {"image/jpeg", familyJPEG},
	".png":  {"image/png", familyPNG},
}

// Descriptor adalah hasil validasi berkas.
type Descriptor struct {
	Extension string
	MediaType string
}

// ValidatePhoto memvalidasi foto absensi.
func ValidatePhoto(fileName, declaredMediaType string, head []byte) (Descriptor, error) {
	return validate(photoTypes, fileName, declaredMediaType, head)
}

// ValidateSupportingDocument memvalidasi dokumen pendukung pengajuan.
func ValidateSupportingDocument(fileName, declaredMediaType string, head []byte) (Descriptor, error) {
	return validate(supportingTypes, fileName, declaredMediaType, head)
}

// ValidateDocument memastikan nama berkas, MIME type yang dilaporkan client, dan signature
// isi berkas konsisten dengan format dokumen karyawan yang disetujui.
func ValidateDocument(fileName, declaredMediaType string, head []byte) (Descriptor, error) {
	return validate(documentTypes, fileName, declaredMediaType, head)
}

func validate(
	allowed map[string]descriptor,
	fileName, declaredMediaType string,
	head []byte,
) (Descriptor, error) {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	expected, ok := allowed[extension]
	if !ok {
		return Descriptor{}, ErrUnsupported
	}
	if declared := normalizeMediaType(declaredMediaType); declared != "" && declared != expected.mediaType {
		return Descriptor{}, ErrUnsupported
	}
	if !matchesSignature(expected.family, head) {
		return Descriptor{}, ErrSignatureMismatch
	}
	return Descriptor{Extension: extension, MediaType: expected.mediaType}, nil
}

func normalizeMediaType(value string) string {
	media, _, _ := strings.Cut(value, ";")
	media = strings.ToLower(strings.TrimSpace(media))
	// Beberapa browser mengirim octet-stream untuk format Office; perlakukan sebagai
	// "tidak dideklarasikan" dan andalkan signature.
	if media == "application/octet-stream" {
		return ""
	}
	return media
}

func matchesSignature(expected family, head []byte) bool {
	switch expected {
	case familyPDF:
		return bytes.HasPrefix(head, []byte("%PDF-"))
	case familyJPEG:
		return bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF})
	case familyPNG:
		return bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	case familyOLE2:
		return bytes.HasPrefix(head, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	case familyOOXML:
		return bytes.HasPrefix(head, []byte{'P', 'K', 0x03, 0x04}) ||
			bytes.HasPrefix(head, []byte{'P', 'K', 0x05, 0x06}) ||
			bytes.HasPrefix(head, []byte{'P', 'K', 0x07, 0x08})
	default:
		return false
	}
}
