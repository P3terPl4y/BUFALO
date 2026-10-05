package exports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Cell struct {
	Value   string
	Numeric bool
}

type Report struct {
	Title   string
	Headers []string
	Rows    [][]Cell
}

var ErrUnsupportedFormat = errors.New("formato de exportación no soportado")

func Render(report Report, format string) ([]byte, string, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		body, err := renderCSV(report)
		return body, "text/csv; charset=utf-8", "csv", err
	case "xlsx", "excel":
		body, err := renderXLSX(report)
		return body, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", err
	case "pdf":
		body, err := renderPDF(report)
		return body, "application/pdf", "pdf", err
	default:
		return nil, "", "", ErrUnsupportedFormat
	}
}

func renderCSV(report Report) ([]byte, error) {
	var out bytes.Buffer
	out.Write([]byte{0xef, 0xbb, 0xbf}) // Excel detects UTF-8 reliably with a BOM.
	w := csv.NewWriter(&out)
	if err := w.Write(report.Headers); err != nil {
		return nil, err
	}
	for _, row := range report.Rows {
		record := make([]string, len(report.Headers))
		for i := range record {
			if i < len(row) {
				record[i] = row[i].Value
				if !row[i].Numeric || !validNumber(row[i].Value) {
					record[i] = csvSafeText(record[i])
				}
			}
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return out.Bytes(), w.Error()
}

func csvSafeText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func renderXLSX(report Report) ([]byte, error) {
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	files := []struct{ name, content string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="` + xmlEscape(sheetName(report.Title)) + `" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`},
		{"xl/worksheets/sheet1.xml", renderWorksheet(report)},
	}
	for _, file := range files {
		entry, err := archive.Create(file.name)
		if err != nil {
			_ = archive.Close()
			return nil, err
		}
		if _, err := entry.Write([]byte(file.content)); err != nil {
			_ = archive.Close()
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func renderWorksheet(report Report) string {
	var out strings.Builder
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	writeRow := func(rowNumber int, cells []Cell) {
		fmt.Fprintf(&out, `<row r="%d">`, rowNumber)
		for column, cell := range cells {
			ref := columnName(column+1) + strconv.Itoa(rowNumber)
			if cell.Numeric && validNumber(cell.Value) {
				fmt.Fprintf(&out, `<c r="%s"><v>%s</v></c>`, ref, cell.Value)
			} else {
				fmt.Fprintf(&out, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, xmlEscape(cell.Value))
			}
		}
		out.WriteString(`</row>`)
	}
	header := make([]Cell, len(report.Headers))
	for i, value := range report.Headers {
		header[i] = Cell{Value: value}
	}
	writeRow(1, header)
	for i, row := range report.Rows {
		cells := make([]Cell, len(report.Headers))
		copy(cells, row)
		writeRow(i+2, cells)
	}
	out.WriteString(`</sheetData></worksheet>`)
	return out.String()
}

func columnName(number int) string {
	var name []byte
	for number > 0 {
		number--
		name = append([]byte{byte('A' + number%26)}, name...)
		number /= 26
	}
	return string(name)
}

func validNumber(value string) bool {
	number, err := strconv.ParseFloat(value, 64)
	return err == nil && !math.IsNaN(number) && !math.IsInf(number, 0)
}

func sheetName(value string) string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\\`, r) || r < 0x20 {
			return '_'
		}
		return r
	}, value))
	if value == "" {
		return "BUFALO"
	}
	if utf8.RuneCountInString(value) > 31 {
		runes := []rune(value)
		value = string(runes[:31])
	}
	return value
}

func xmlEscape(value string) string {
	var out bytes.Buffer
	clean := strings.Map(func(r rune) rune {
		if (r < 0x20 && r != '\t' && r != '\n' && r != '\r') || (r >= 0x7f && r <= 0x9f) {
			return '\uFFFD'
		}
		return r
	}, value)
	_ = xml.EscapeText(&out, []byte(clean))
	return out.String()
}

func renderPDF(report Report) ([]byte, error) {
	lines := []string{report.Title, strings.Join(report.Headers, " | ")}
	for _, row := range report.Rows {
		values := make([]string, len(report.Headers))
		for i := range values {
			if i < len(row) {
				values[i] = row[i].Value
			}
		}
		lines = append(lines, strings.Join(values, " | "))
	}
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapPDFLine(line, 110)...)
	}
	lines = wrapped
	const linesPerPage = 48
	pageCount := (len(lines) + linesPerPage - 1) / linesPerPage
	if pageCount == 0 {
		pageCount = 1
	}
	objects := make([][]byte, 3+pageCount*2)
	objects[0] = []byte("<< /Type /Catalog /Pages 2 0 R >>")
	fontID := 3
	pageIDs := make([]string, 0, pageCount)
	for page := 0; page < pageCount; page++ {
		pageID := 4 + page*2
		contentID := pageID + 1
		pageIDs = append(pageIDs, fmt.Sprintf("%d 0 R", pageID))
		start := page * linesPerPage
		end := start + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		var stream strings.Builder
		stream.WriteString("BT /F1 9 Tf 40 800 Td 13 TL\n")
		for _, line := range lines[start:end] {
			stream.WriteByte('<')
			stream.WriteString(hex.EncodeToString(winAnsi(line)))
			stream.WriteString("> Tj T*\n")
		}
		stream.WriteString("ET")
		streamBytes := []byte(stream.String())
		objects[pageID-1] = []byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 842] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", fontID, contentID))
		objects[contentID-1] = []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(streamBytes), streamBytes))
	}
	objects[1] = []byte("<< /Type /Pages /Kids [" + strings.Join(pageIDs, " ") + fmt.Sprintf("] /Count %d >>", pageCount))
	objects[2] = []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	return buildPDF(objects), nil
}

func wrapPDFLine(line string, width int) []string {
	if width < 1 {
		width = 110
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	wrapped := make([]string, 0, 1)
	current := ""
	for _, word := range words {
		for utf8.RuneCountInString(word) > width {
			runes := []rune(word)
			if current != "" {
				wrapped = append(wrapped, current)
				current = ""
			}
			wrapped = append(wrapped, string(runes[:width]))
			word = string(runes[width:])
		}
		if current == "" {
			current = word
		} else if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
			current += " " + word
		} else {
			wrapped = append(wrapped, current)
			current = word
		}
	}
	if current != "" {
		wrapped = append(wrapped, current)
	}
	return wrapped
}

func buildPDF(objects [][]byte) []byte {
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(object)
		out.WriteString("\nendobj\n")
	}
	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefOffset)
	return out.Bytes()
}

func winAnsi(value string) []byte {
	bytesOut := make([]byte, 0, len(value))
	for _, r := range value {
		switch {
		case r >= 0x20 && r <= 0x7e, r >= 0xa0 && r <= 0xff:
			bytesOut = append(bytesOut, byte(r))
		case r == '\n' || r == '\r' || r == '\t':
			bytesOut = append(bytesOut, ' ')
		default:
			mapped, ok := winAnsiSpecial[r]
			if ok {
				bytesOut = append(bytesOut, mapped)
			} else {
				bytesOut = append(bytesOut, '?')
			}
		}
	}
	return bytesOut
}

var winAnsiSpecial = map[rune]byte{
	'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86,
	'‡': 0x87, 'ˆ': 0x88, '‰': 0x89, 'Š': 0x8a, '‹': 0x8b, 'Œ': 0x8c,
	'Ž': 0x8e, '‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95,
	'–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9a, '›': 0x9b,
	'œ': 0x9c, 'ž': 0x9e, 'Ÿ': 0x9f,
}

func Number(value int64) Cell { return Cell{Value: strconv.FormatInt(value, 10), Numeric: true} }
