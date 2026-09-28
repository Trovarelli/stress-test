package report_test

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jtnet/stress-test/internal/report"
)

func TestReport_RecordResult_And_Render(t *testing.T) {
	rep := report.NewReport("http://example.com", 100, 10)
	rep.TotalDuration = 1500 * time.Millisecond

	// Record 80 OKs
	for i := 0; i < 80; i++ {
		rep.RecordResult(http.StatusOK, nil)
	}

	// Record 15 404s
	for i := 0; i < 15; i++ {
		rep.RecordResult(http.StatusNotFound, nil)
	}

	// Record 3 500s
	for i := 0; i < 3; i++ {
		rep.RecordResult(http.StatusInternalServerError, nil)
	}

	// Record 2 Network Errors
	for i := 0; i < 2; i++ {
		rep.RecordResult(0, errors.New("connection reset"))
	}

	var buf bytes.Buffer
	rep.Render(&buf)
	output := buf.String()

	// Assertions on required fields
	if !strings.Contains(output, "http://example.com") {
		t.Errorf("expected output to contain URL, got:\n%s", output)
	}
	if !strings.Contains(output, "Requisições Solicitadas        : 100") {
		t.Errorf("expected requested requests = 100, got:\n%s", output)
	}
	if !strings.Contains(output, "Requisições Executadas         : 100") {
		t.Errorf("expected executed requests = 100, got:\n%s", output)
	}
	if !strings.Contains(output, "Requisições com Status 200 (OK) : 80") {
		t.Errorf("expected 200 OK count = 80, got:\n%s", output)
	}
	if !strings.Contains(output, "[HTTP 404 - Not Found] : 15") {
		t.Errorf("expected 404 count = 15, got:\n%s", output)
	}
	if !strings.Contains(output, "[HTTP 500 - Internal Server Error] : 3") {
		t.Errorf("expected 500 count = 3, got:\n%s", output)
	}
	if !strings.Contains(output, "Falhas de Rede/Conexão         : 2") {
		t.Errorf("expected 2 network errors, got:\n%s", output)
	}
}

func TestReport_Render_NoOtherCodes(t *testing.T) {
	rep := report.NewReport("https://allok.com", 10, 2)
	rep.TotalDuration = 500 * time.Millisecond

	for i := 0; i < 10; i++ {
		rep.RecordResult(http.StatusOK, nil)
	}

	var buf bytes.Buffer
	rep.Render(&buf)
	output := buf.String()

	if !strings.Contains(output, "Nenhum outro código HTTP retornado.") {
		t.Errorf("expected 'Nenhum outro código HTTP retornado.', got:\n%s", output)
	}
	if strings.Contains(output, "Falhas de Rede/Conexão") {
		t.Errorf("did not expect network errors section, got:\n%s", output)
	}
}
