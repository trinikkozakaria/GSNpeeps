package export

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PDFContentType adalah media type PDF sesuai kontrak export.
const PDFContentType = "application/pdf"

const (
	pageWidth    = 842.0 // A4 landscape
	pageHeight   = 595.0
	marginLeft   = 36.0
	marginTop    = 40.0
	lineHeight   = 14.0
	bodyFontSize = 9.0
	rowsPerPage  = 34
)

// WritePDF menulis tabel sebagai dokumen PDF sederhana tanpa watermark, memakai core font
// Helvetica sehingga tidak ada berkas font yang perlu di-embed.
func WritePDF(writer io.Writer, table Table) error {
	pages := paginate(table)
	// Object 1 catalog, 2 pages, 3 font, lalu tiap halaman memakai dua object
	// (page + content stream).
	objects := make([]string, 0, 3+len(pages)*2)

	pageIDs := make([]int, len(pages))
	for index := range pages {
		pageIDs[index] = 4 + index*2
	}

	references := make([]string, 0, len(pageIDs))
	for _, id := range pageIDs {
		references = append(references, strconv.Itoa(id)+" 0 R")
	}

	objects = append(objects, "<< /Type /Catalog /Pages 2 0 R >>")
	objects = append(objects, fmt.Sprintf(
		"<< /Type /Pages /Kids [%s] /Count %d >>",
		strings.Join(references, " "), len(pages),
	))
	objects = append(objects, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")

	for index, page := range pages {
		contentID := pageIDs[index] + 1
		objects = append(objects, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] "+
				"/Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, contentID,
		))
		stream := pageStream(table, page, index+1, len(pages))
		objects = append(objects, fmt.Sprintf(
			"<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream,
		))
	}

	var document bytes.Buffer
	document.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = document.Len()
		document.WriteString(strconv.Itoa(index+1) + " 0 obj\n" + object + "\nendobj\n")
	}

	xrefOffset := document.Len()
	document.WriteString("xref\n0 " + strconv.Itoa(len(objects)+1) + "\n")
	document.WriteString("0000000000 65535 f \n")
	for index := 1; index <= len(objects); index++ {
		document.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[index]))
	}
	document.WriteString(fmt.Sprintf(
		"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xrefOffset,
	))

	if _, err := writer.Write(document.Bytes()); err != nil {
		return fmt.Errorf("write pdf document: %w", err)
	}
	return nil
}

func paginate(table Table) [][][]string {
	if len(table.Rows) == 0 {
		return [][][]string{{}}
	}
	pages := make([][][]string, 0, len(table.Rows)/rowsPerPage+1)
	for start := 0; start < len(table.Rows); start += rowsPerPage {
		end := min(start+rowsPerPage, len(table.Rows))
		pages = append(pages, table.Rows[start:end])
	}
	return pages
}

func pageStream(table Table, rows [][]string, pageNumber, pageCount int) string {
	columnWidth := (pageWidth - 2*marginLeft) / float64(max(len(table.Headers), 1))
	var stream bytes.Buffer

	stream.WriteString("BT\n/F1 12 Tf\n")
	stream.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", marginLeft, pageHeight-marginTop))
	stream.WriteString("(" + escapePDFText(SanitizeCell(table.Title)) + ") Tj\nET\n")

	stream.WriteString("BT\n/F1 8 Tf\n")
	stream.WriteString(fmt.Sprintf("1 0 0 1 %.2f %.2f Tm\n", pageWidth-marginLeft-80, pageHeight-marginTop))
	stream.WriteString(fmt.Sprintf("(Halaman %d dari %d) Tj\nET\n", pageNumber, pageCount))

	writeTextRow := func(values []string, y float64, size float64) {
		for column, value := range values {
			if column >= len(table.Headers) {
				break
			}
			x := marginLeft + float64(column)*columnWidth
			stream.WriteString(fmt.Sprintf("BT\n/F1 %.1f Tf\n1 0 0 1 %.2f %.2f Tm\n", size, x, y))
			text := truncate(SanitizeCell(value), int(columnWidth/(size*0.5)))
			stream.WriteString("(" + escapePDFText(text) + ") Tj\nET\n")
		}
	}

	headerY := pageHeight - marginTop - 2*lineHeight
	writeTextRow(table.Headers, headerY, bodyFontSize)
	stream.WriteString(fmt.Sprintf(
		"%.2f %.2f m %.2f %.2f l S\n",
		marginLeft, headerY-4, pageWidth-marginLeft, headerY-4,
	))

	for index, row := range rows {
		writeTextRow(row, headerY-float64(index+1)*lineHeight, bodyFontSize)
	}
	return stream.String()
}

func truncate(value string, limit int) string {
	if limit < 4 {
		limit = 4
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-3]) + "..."
}

// escapePDFText melindungi karakter sintaks PDF dan mengganti rune non-ASCII dengan spasi
// agar stream tetap valid pada encoding WinAnsi core font.
func escapePDFText(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r == '(' || r == ')' || r == '\\':
			builder.WriteByte('\\')
			builder.WriteRune(r)
		case r < 0x20 || r > 0x7e:
			builder.WriteByte(' ')
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
