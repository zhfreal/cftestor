package ping

import (
	"net/http"
	"testing"
	"time"

	"cftestor/internal/config"
	utls "github.com/refraction-networking/utls"
)

func TestNewHttpClientWithUTLSPresets(t *testing.T) {
	presets := []struct {
		name    string
		helloID utls.ClientHelloID
	}{
		{"Chrome", utls.HelloChrome_Auto},
		{"Firefox", utls.HelloFirefox_Auto},
		{"Safari", utls.HelloSafari_Auto},
		{"Edge", utls.HelloEdge_Auto},
	}

	for _, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			client, tr := NewHttpClient(p.helloID, "1.1.1.1:443", 5*time.Second)
			if client == nil || tr == nil {
				t.Fatalf("NewHttpClient returned nil client or transport for %s", p.name)
			}
			if tr.clientHello.Client != p.helloID.Client {
				t.Errorf("ClientHello mismatch: expected %v, got %v", p.helloID, tr.clientHello)
			}
			tr.CloseIdleConnections()
		})
	}
}

func TestGetMaxFailureCalculations(t *testing.T) {
	config.Config.DTCount = 4
	config.Config.DLTCount = 1
	config.Config.EnableDTEvaluation = false
	config.Config.DTEvaluationDTPR = 100.0

	// When evaluation is disabled
	if maxF := GetMaxFailure(true); maxF != 4 {
		t.Errorf("expected max DT failure 4, got %d", maxF)
	}
	if maxF := GetMaxFailure(false); maxF != 1 {
		t.Errorf("expected max DLT failure 1, got %d", maxF)
	}

	// When evaluation is enabled
	config.Config.EnableDTEvaluation = true
	config.Config.DTEvaluationDTPR = 75.0 // allow 1 failure out of 4 (round(4 * 0.25) = 1)
	if maxEvF := GetMaxEvDTFailure(); maxEvF != 1 {
		t.Errorf("expected max Ev DT failure 1 for 75%% DTPR of 4 attempts, got %d", maxEvF)
	}
}

func TestUTLSTransportHTTPRouting(t *testing.T) {
	tr := NewUTLSTransport(utls.HelloChrome_Auto, "127.0.0.1:80", 5*time.Second)
	defer tr.CloseIdleConnections()

	httpReq, err := http.NewRequest("GET", "http://127.0.0.1:80", nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	// Should route via tr1
	_, _ = tr.RoundTrip(httpReq)
}
