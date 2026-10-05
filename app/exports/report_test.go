package exports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestRenderCSVProtectsSpreadsheetFormulasAndKeepsUTF8(t *testing.T) {
	body, contentType, extension, err := Render(Report{
		Headers: []string{"Nombre", "Importe"},
		Rows: [][]Cell{
			{{Value: "=HYPERLINK(\"https://bad\")"}, {Value: "12.50", Numeric: true}},
			{{Value: "@SUM(A1:A2)", Numeric: true}, {Value: "-4.00", Numeric: true}},
		},
	}, "csv")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "text/csv; charset=utf-8" || extension != "csv" {
		t.Fatalf("unexpected response metadata: %q %q", contentType, extension)
	}
	if !bytes.HasPrefix(body, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("CSV should include a UTF-8 BOM")
	}
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})))
	if _, err := reader.Read(); err != nil {
		t.Fatal(err)
	}
	row, err := reader.Read()
	if err != nil {
		t.Fatal(err)
	}
	if row[0] != "'=HYPERLINK(\"https://bad\")" || row[1] != "12.50" {
		t.Fatalf("unexpected escaped CSV row: %#v", row)
	}
	row, err = reader.Read()
	if err != nil || row[0] != "'@SUM(A1:A2)" || row[1] != "-4.00" {
		t.Fatalf("invalid numeric flag bypassed formula sanitization: row=%#v error=%v", row, err)
	}
}

func TestRenderExcelXMLAndPDF(t *testing.T) {
	report := Report{Title: "Métricas BUFALO", Headers: []string{"Estado", "Total"}, Rows: [][]Cell{{{Value: "pagada"}, Number(2)}}}
	workbook, contentType, extension, err := Render(report, "excel")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" || extension != "xlsx" || !bytes.HasPrefix(workbook, []byte("PK")) {
		t.Fatal("XLSX export metadata or ZIP signature is incorrect")
	}
	archive, err := zip.NewReader(bytes.NewReader(workbook), int64(len(workbook)))
	if err != nil {
		t.Fatalf("XLSX ZIP is malformed: %v", err)
	}
	var worksheet []byte
	for _, file := range archive.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			opened, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			worksheet, err = io.ReadAll(opened)
			opened.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(worksheet) == 0 || !bytes.Contains(worksheet, []byte(`<c r="B2"><v>2</v></c>`)) {
		t.Fatal("XLSX worksheet missing a numeric cell")
	}
	if err := xml.Unmarshal(worksheet, &struct{}{}); err != nil {
		t.Fatalf("XLSX worksheet XML is malformed: %v", err)
	}
	pdf, contentType, extension, err := Render(report, "pdf")
	if err != nil {
		t.Fatal(err)
	}
	pdfSpanishTitle := "<" + hex.EncodeToString(winAnsi("Métricas BUFALO")) + "> Tj"
	if contentType != "application/pdf" || extension != "pdf" || !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) || !bytes.HasSuffix(pdf, []byte("%%EOF")) || !bytes.Contains(pdf, []byte(pdfSpanishTitle)) {
		t.Fatal("PDF export is malformed or lost supported Spanish characters")
	}
	marker := []byte("startxref\n")
	index := bytes.LastIndex(pdf, marker)
	if index < 0 {
		t.Fatal("PDF cross-reference offset is missing")
	}
	end := bytes.IndexByte(pdf[index+len(marker):], '\n')
	xref, err := strconv.Atoi(strings.TrimSpace(string(pdf[index+len(marker) : index+len(marker)+end])))
	if err != nil || xref >= len(pdf) || string(pdf[xref:xref+4]) != "xref" {
		t.Fatal("PDF cross-reference offset does not point to the xref table")
	}

	manyRows := make([][]Cell, 50)
	for i := range manyRows {
		manyRows[i] = []Cell{{Value: "fila"}, Number(int64(i))}
	}
	multiPagePDF, _, _, err := Render(Report{Title: "Prueba", Headers: []string{"Nombre", "Cantidad"}, Rows: manyRows}, "pdf")
	if err != nil || !bytes.Contains(multiPagePDF, []byte("/Count 2 >>")) {
		t.Fatalf("PDF should paginate long reports; error=%v", err)
	}
}

func TestRenderRejectsUnknownFormat(t *testing.T) {
	if _, _, _, err := Render(Report{}, "ods"); err != ErrUnsupportedFormat {
		t.Fatalf("unknown format error = %v", err)
	}
}
