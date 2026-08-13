package filetype

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pdfHead   = []byte("%PDF-1.7\n")
	pngHead   = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	jpegHead  = []byte{0xFF, 0xD8, 0xFF, 0xE0}
	ole2Head  = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	ooxmlHead = []byte{'P', 'K', 0x03, 0x04}
)

func TestValidateDocumentAcceptsApprovedFormats(t *testing.T) {
	cases := []struct {
		fileName  string
		declared  string
		head      []byte
		mediaType string
	}{
		{"ijazah.pdf", "application/pdf", pdfHead, "application/pdf"},
		{"foto.jpg", "image/jpeg", jpegHead, "image/jpeg"},
		{"foto.jpeg", "", jpegHead, "image/jpeg"},
		{"scan.png", "image/png", pngHead, "image/png"},
		{"surat.doc", "application/msword", ole2Head, "application/msword"},
		{"data.xls", "application/vnd.ms-excel", ole2Head, "application/vnd.ms-excel"},
		{"materi.ppt", "application/vnd.ms-powerpoint", ole2Head, "application/vnd.ms-powerpoint"},
		{
			"surat.docx",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			ooxmlHead,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{"data.xlsx", "application/octet-stream", ooxmlHead,
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"materi.pptx", "", ooxmlHead,
			"application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"HURUF-BESAR.PDF", "", pdfHead, "application/pdf"},
	}

	for _, testCase := range cases {
		t.Run(testCase.fileName, func(t *testing.T) {
			descriptor, err := ValidateDocument(testCase.fileName, testCase.declared, testCase.head)

			require.NoError(t, err)
			assert.Equal(t, testCase.mediaType, descriptor.MediaType)
		})
	}
}

func TestValidateDocumentRejectsArchives(t *testing.T) {
	// DOCX secara fisik adalah container ZIP; extension arsip tetap harus ditolak.
	_, err := ValidateDocument("arsip.zip", "application/zip", ooxmlHead)
	require.ErrorIs(t, err, ErrUnsupported)

	_, err = ValidateDocument("arsip.rar", "application/vnd.rar", []byte("Rar!\x1a\x07\x00"))
	require.ErrorIs(t, err, ErrUnsupported)
}

func TestValidateDocumentRejectsUnknownExtension(t *testing.T) {
	_, err := ValidateDocument("skrip.sh", "text/x-shellscript", []byte("#!/bin/sh\n"))
	require.ErrorIs(t, err, ErrUnsupported)

	_, err = ValidateDocument("tanpa-extension", "application/pdf", pdfHead)
	require.ErrorIs(t, err, ErrUnsupported)
}

func TestValidateDocumentRejectsSignatureMismatch(t *testing.T) {
	// Berkas berbahaya yang dinamai ulang menjadi PDF harus gagal pada pemeriksaan signature.
	_, err := ValidateDocument("penyamaran.pdf", "application/pdf", []byte("MZ\x90\x00 executable"))
	require.ErrorIs(t, err, ErrSignatureMismatch)

	_, err = ValidateDocument("gambar.png", "image/png", jpegHead)
	require.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestValidateDocumentRejectsMismatchedDeclaredMediaType(t *testing.T) {
	_, err := ValidateDocument("ijazah.pdf", "image/png", pdfHead)
	require.ErrorIs(t, err, ErrUnsupported)
}

func TestValidateDocumentRejectsEmptyContent(t *testing.T) {
	_, err := ValidateDocument("ijazah.pdf", "application/pdf", nil)
	require.ErrorIs(t, err, ErrSignatureMismatch)
}
