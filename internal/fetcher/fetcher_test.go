package fetcher

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestParseCIDRsAndContains(t *testing.T) {
	cidrs := []string{
		"104.16.0.0/16",
		"172.64.0.0/13",
		"2606:4700::/32",
		"invalid-cidr",
	}

	list := parseCIDRs(cidrs)
	if len(list) != 3 {
		t.Fatalf("expected 3 valid parsed CIDRs, got %d", len(list))
	}

	// Test IPv4 containment
	if !list.Contains(net.ParseIP("104.16.1.1")) {
		t.Errorf("expected 104.16.1.1 to be contained in 104.16.0.0/16")
	}
	if !list.Contains(net.ParseIP("172.65.10.20")) {
		t.Errorf("expected 172.65.10.20 to be contained in 172.64.0.0/13")
	}
	if list.Contains(net.ParseIP("8.8.8.8")) {
		t.Errorf("8.8.8.8 should not be in the list")
	}

	// Test IPv6 containment
	if !list.Contains(net.ParseIP("2606:4700:100::1")) {
		t.Errorf("expected 2606:4700:100::1 to be contained in 2606:4700::/32")
	}
	if list.Contains(net.ParseIP("2001:4860:4860::8888")) {
		t.Errorf("2001:4860:4860::8888 should not be in the list")
	}
}

func TestMiekgDNSMessageEncodingAndDecoding(t *testing.T) {
	// Test miekg/dns message packing and unpacking
	m := new(dns.Msg)
	m.SetQuestion("cloudflare.com.", dns.TypeA)
	m.RecursionDesired = true

	packed, err := m.Pack()
	if err != nil {
		t.Fatalf("dns.Msg.Pack failed: %v", err)
	}

	unpacked := new(dns.Msg)
	if err := unpacked.Unpack(packed); err != nil {
		t.Fatalf("dns.Msg.Unpack failed: %v", err)
	}

	if len(unpacked.Question) != 1 {
		t.Fatalf("expected 1 question, got %d", len(unpacked.Question))
	}
	if unpacked.Question[0].Name != "cloudflare.com." || unpacked.Question[0].Qtype != dns.TypeA {
		t.Errorf("question mismatch: %+v", unpacked.Question[0])
	}
}

func TestQueryDoHWithMockServer(t *testing.T) {
	// Create mock DoH HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/dns-message" {
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}

		reqMsg := new(dns.Msg)
		if err := reqMsg.Unpack(body); err != nil {
			http.Error(w, "bad dns packet", http.StatusBadRequest)
			return
		}

		respMsg := new(dns.Msg)
		respMsg.SetReply(reqMsg)
		respMsg.Authoritative = true

		for _, q := range reqMsg.Question {
			if q.Qtype == dns.TypeA {
				rr, _ := dns.NewRR(q.Name + " 300 IN A 104.16.1.1")
				respMsg.Answer = append(respMsg.Answer, rr)
			} else if q.Qtype == dns.TypeAAAA {
				rr, _ := dns.NewRR(q.Name + " 300 IN AAAA 2606:4700::1")
				respMsg.Answer = append(respMsg.Answer, rr)
			}
		}

		respBytes, err := respMsg.Pack()
		if err != nil {
			http.Error(w, "pack error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))
	defer server.Close()

	// Build a DoH request to our mock server using miekg/dns
	msg := new(dns.Msg)
	msg.SetQuestion("test.cloudflare.com.", dns.TypeA)
	msg.RecursionDesired = true
	packed, err := msg.Pack()
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	req, err := http.NewRequest("POST", server.URL, bytesNewReader(packed))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}

	respMsg := new(dns.Msg)
	if err := respMsg.Unpack(respBody); err != nil {
		t.Fatalf("Unpack response failed: %v", err)
	}

	if len(respMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(respMsg.Answer))
	}
	aRecord, ok := respMsg.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected *dns.A record, got %T", respMsg.Answer[0])
	}
	if aRecord.A.String() != "104.16.1.1" {
		t.Errorf("expected IP 104.16.1.1, got %s", aRecord.A.String())
	}
}

func bytesNewReader(b []byte) io.Reader {
	return &bytesReader{b: b, i: 0}
}

type bytesReader struct {
	b []byte
	i int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n = copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func TestRipeStatJSONDecoding(t *testing.T) {
	rawJSON := `{
		"data": {
			"prefixes": [
				{"prefix": "104.16.0.0/13"},
				{"prefix": "104.24.0.0/14"},
				{"prefix": "2606:4700::/32"}
			]
		}
	}`

	var data ripeStatResponse
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(data.Data.Prefixes) != 3 {
		t.Fatalf("expected 3 prefixes, got %d", len(data.Data.Prefixes))
	}
	if data.Data.Prefixes[0].Prefix != "104.16.0.0/13" {
		t.Errorf("prefix 0 mismatch: %s", data.Data.Prefixes[0].Prefix)
	}
}

func TestLoadECHConfig(t *testing.T) {
	// 1. Direct base64 string
	bytes, err := LoadECHConfig(DefaultCloudflareECHConfigBase64, "1.1.1.1:53", "cloudflare-ech.com")
	if err != nil {
		t.Fatalf("LoadECHConfig with base64 failed: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatalf("expected non-empty ECH config bytes")
	}

	// 2. From temporary file
	tmpFile, err := os.CreateTemp(t.TempDir(), "ech-*.txt")
	if err != nil {
		t.Fatalf("create temp file failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(DefaultCloudflareECHConfigBase64); err != nil {
		t.Fatalf("write to temp file failed: %v", err)
	}
	tmpFile.Close()

	fileBytes, err := LoadECHConfig(tmpFile.Name(), "1.1.1.1:53", "cloudflare-ech.com")
	if err != nil {
		t.Fatalf("LoadECHConfig with file failed: %v", err)
	}
	if len(fileBytes) != len(bytes) {
		t.Errorf("expected len %d, got %d", len(bytes), len(fileBytes))
	}

	// 3. Invalid base64
	_, err = LoadECHConfig("invalid!base64!", "1.1.1.1:53", "cloudflare-ech.com")
	if err == nil {
		t.Errorf("expected error for invalid base64")
	}

	// 4. Fallback on invalid DNS server for default domain
	fbBytes, err := LoadECHConfig("", "127.0.0.1:65530", "cloudflare-ech.com")
	if err != nil {
		t.Fatalf("LoadECHConfig fallback failed: %v", err)
	}
	if len(fbBytes) != len(bytes) {
		t.Errorf("expected fallback bytes to match embedded base64")
	}

	// 5. Custom domain with invalid DNS server should fail
	_, err = LoadECHConfig("", "127.0.0.1:65530", "custom-domain.example.com")
	if err == nil {
		t.Errorf("expected error for custom domain with invalid DNS server")
	}

	// 5b. Default domain with uppercase and trailing dot should still fallback
	fbBytesCase, err := LoadECHConfig("", "127.0.0.1:65530", "Cloudflare-ECH.com.")
	if err != nil || len(fbBytesCase) != len(bytes) {
		t.Errorf("expected fallback for Cloudflare-ECH.com. to succeed, got %v", err)
	}

	// 6. Raw binary ECH file
	binFile, err := os.CreateTemp(t.TempDir(), "ech-*.bin")
	if err != nil {
		t.Fatalf("create temp bin file failed: %v", err)
	}
	defer os.Remove(binFile.Name())
	if _, err := binFile.Write(bytes); err != nil {
		t.Fatalf("write to temp bin file failed: %v", err)
	}
	binFile.Close()

	binBytes, err := LoadECHConfig(binFile.Name(), "1.1.1.1:53", "cloudflare-ech.com")
	if err != nil {
		t.Fatalf("LoadECHConfig with bin file failed: %v", err)
	}
	if len(binBytes) != len(bytes) {
		t.Errorf("expected len %d, got %d", len(bytes), len(binBytes))
	}

	// 7. Unpadded base64 string
	unpadded := strings.TrimRight(DefaultCloudflareECHConfigBase64, "=")
	unpaddedBytes, err := LoadECHConfig(unpadded, "1.1.1.1:53", "cloudflare-ech.com")
	if err != nil {
		t.Fatalf("LoadECHConfig with unpadded base64 failed: %v", err)
	}
	if len(unpaddedBytes) != len(bytes) {
		t.Errorf("expected len %d, got %d", len(bytes), len(unpaddedBytes))
	}
}

func TestValidateECHConfigList(t *testing.T) {
	// Too short (< 4 bytes)
	if err := ValidateECHConfigList([]byte{0x00, 0x01}); err == nil {
		t.Errorf("expected error for short ECH bytes")
	}

	// Invalid length header
	if err := ValidateECHConfigList([]byte{0x00, 0x10, 0x00, 0x01}); err == nil {
		t.Errorf("expected error for mismatched length header")
	}

	// Valid header
	validBytes := []byte{0x00, 0x02, 0xfe, 0x0d}
	if err := ValidateECHConfigList(validBytes); err != nil {
		t.Errorf("expected valid for proper length header, got: %v", err)
	}
}
