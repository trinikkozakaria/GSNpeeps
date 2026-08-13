package export

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeCellNeutralizesFormulaTriggers(t *testing.T) {
	cases := map[string]string{
		"=1+1":                    "'=1+1",
		"+CMD|' /C calc'!A0":      "'+CMD|' /C calc'!A0",
		"-2+3":                    "'-2+3",
		"@SUM(A1)":                "'@SUM(A1)",
		"\tterindentasi":          "' terindentasi",
		"Karyawan Uji":            "Karyawan Uji",
		"EMP-001":                 "EMP-001",
		"employee@example.test":   "employee@example.test",
		"nilai\x00dengan-kontrol": "nilaidengan-kontrol",
	}

	for input, expected := range cases {
		assert.Equal(t, expected, SanitizeCell(input), "input %q", input)
	}
}

func TestSanitizeFileNameRemovesPathSeparators(t *testing.T) {
	assert.Equal(t, "etc-passwd", SanitizeFileName("../../etc/passwd"))
	assert.Equal(t, "karyawan-20260801.xlsx", SanitizeFileName("karyawan-20260801.xlsx"))
	assert.Equal(t, "export", SanitizeFileName("///"))
}

func TestWriteXLSXProducesReadableWorkbook(t *testing.T) {
	var buffer bytes.Buffer

	require.NoError(t, WriteXLSX(&buffer, sampleTable()))

	archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)

	names := make([]string, 0, len(archive.File))
	for _, entry := range archive.File {
		names = append(names, entry.Name)
	}
	assert.Contains(t, names, "[Content_Types].xml")
	assert.Contains(t, names, "xl/workbook.xml")
	assert.Contains(t, names, "xl/worksheets/sheet1.xml")

	sheet := readEntry(t, archive, "xl/worksheets/sheet1.xml")
	assert.Contains(t, sheet, "NIP")
	assert.Contains(t, sheet, "Karyawan Uji")
	assert.Contains(t, sheet, `r="A1"`)
	assert.Contains(t, sheet, `r="F2"`)
	// Karakter XML khusus harus di-escape, bukan merusak dokumen.
	assert.Contains(t, sheet, "&amp;")
	assert.NotContains(t, sheet, "Riset & Pengembangan")
}

func TestWriteXLSXColumnNamesPassSingleLetterBoundary(t *testing.T) {
	assert.Equal(t, "A", columnName(0))
	assert.Equal(t, "Z", columnName(25))
	assert.Equal(t, "AA", columnName(26))
	assert.Equal(t, "AB", columnName(27))
}

func TestWritePDFProducesValidDocumentStructure(t *testing.T) {
	var buffer bytes.Buffer

	require.NoError(t, WritePDF(&buffer, sampleTable()))

	content := buffer.String()
	assert.True(t, strings.HasPrefix(content, "%PDF-1.4"))
	assert.Contains(t, content, "/Type /Catalog")
	assert.Contains(t, content, "/Type /Page")
	assert.Contains(t, content, "startxref")
	assert.True(t, strings.HasSuffix(content, "%%EOF\n"))
	// Tidak ada watermark yang ditambahkan pada berkas export.
	assert.NotContains(t, strings.ToLower(content), "watermark")
}

func TestWritePDFPaginatesLargeDataset(t *testing.T) {
	table := sampleTable()
	table.Rows = make([][]string, 0, rowsPerPage*2+5)
	for index := 0; index < rowsPerPage*2+5; index++ {
		table.Rows = append(table.Rows, []string{"EMP", "Nama", "surel", "Dept", "Jabatan", "aktif"})
	}

	var buffer bytes.Buffer
	require.NoError(t, WritePDF(&buffer, table))

	assert.Equal(t, 3, strings.Count(buffer.String(), "/Type /Page\n")+
		strings.Count(buffer.String(), "/Type /Page /Parent"))
	assert.Contains(t, buffer.String(), "(Halaman 3 dari 3) Tj")
}

func TestWritePDFEscapesSyntaxCharacters(t *testing.T) {
	table := sampleTable()
	table.Rows = [][]string{{"EMP-001", `Nama (Berkurung) \ Uji`, "a@b.test", "Dept", "Jabatan", "aktif"}}

	var buffer bytes.Buffer
	require.NoError(t, WritePDF(&buffer, table))

	assert.Contains(t, buffer.String(), `\(Berkurung\)`)
}

func sampleTable() Table {
	return Table{
		Title:   "GSNpeeps - Data Karyawan",
		Headers: []string{"NIP", "Nama", "Email", "Departemen", "Jabatan", "Status"},
		Rows: [][]string{{
			"EMP-001",
			"Karyawan Uji",
			"employee@example.test",
			"Riset & Pengembangan",
			"Staff",
			"aktif",
		}},
	}
}

func readEntry(t *testing.T, archive *zip.Reader, name string) string {
	t.Helper()
	for _, entry := range archive.File {
		if entry.Name != name {
			continue
		}
		reader, err := entry.Open()
		require.NoError(t, err)
		defer reader.Close()
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		return string(content)
	}
	t.Fatalf("entry %s tidak ditemukan", name)
	return ""
}
