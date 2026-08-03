package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// XLSXContentType adalah media type SpreadsheetML sesuai kontrak export.
const XLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// Table adalah dataset export: satu baris header diikuti baris data. Seluruh nilai berupa
// string agar spreadsheet tidak menafsirkan ulang NIP atau nomor identitas sebagai angka.
type Table struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// WriteXLSX menulis tabel sebagai workbook XLSX minimal ke writer. Setiap sel ditulis
// sebagai inline string yang sudah dinetralkan dari formula injection.
func WriteXLSX(writer io.Writer, table Table) error {
	archive := zip.NewWriter(writer)
	files := []struct {
		name    string
		content string
	}{
		{"[Content_Types].xml", contentTypesXML},
		{"_rels/.rels", rootRelsXML},
		{"xl/workbook.xml", workbookXML},
		{"xl/_rels/workbook.xml.rels", workbookRelsXML},
		{"xl/worksheets/sheet1.xml", sheetXML(table)},
	}
	for _, file := range files {
		entry, err := archive.Create(file.name)
		if err != nil {
			return fmt.Errorf("create xlsx entry %s: %w", file.name, err)
		}
		if _, err := io.WriteString(entry, file.content); err != nil {
			return fmt.Errorf("write xlsx entry %s: %w", file.name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("finalize xlsx archive: %w", err)
	}
	return nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
	`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
	`<Default Extension="xml" ContentType="application/xml"/>` +
	`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
	`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
	`</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
	`</Relationships>`

const workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
	`<sheets><sheet name="Data" sheetId="1" r:id="rId1"/></sheets></workbook>`

const workbookRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
	`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
	`</Relationships>`

func sheetXML(table Table) string {
	var builder bytes.Buffer
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	writeRow(&builder, 1, table.Headers)
	for index, row := range table.Rows {
		writeRow(&builder, index+2, row)
	}
	builder.WriteString(`</sheetData></worksheet>`)
	return builder.String()
}

func writeRow(builder *bytes.Buffer, rowNumber int, values []string) {
	builder.WriteString(`<row r="` + strconv.Itoa(rowNumber) + `">`)
	for column, value := range values {
		reference := columnName(column) + strconv.Itoa(rowNumber)
		builder.WriteString(`<c r="` + reference + `" t="inlineStr"><is><t xml:space="preserve">`)
		builder.WriteString(escapeXML(SanitizeCell(value)))
		builder.WriteString(`</t></is></c>`)
	}
	builder.WriteString(`</row>`)
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}

func escapeXML(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		return ""
	}
	return buffer.String()
}
