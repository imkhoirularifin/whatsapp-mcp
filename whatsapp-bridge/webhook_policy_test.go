package main

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

var testSelf = netip.MustParseAddrPort("127.0.0.1:8080")

// testAllowLocal is the allowlist harnesses use when exercising local webhook receivers.
var testAllowLocal = parseAllowedHosts("127.0.0.1, localhost")

func TestValidateWebhookURL(t *testing.T) {
	denyAll := urlPolicy{self: testSelf}
	local := urlPolicy{self: testSelf, allowedHosts: testAllowLocal}
	allow := urlPolicy{self: testSelf, allowedHosts: parseAllowedHosts("n8n.example.org, 127.0.0.1, *.hooks.example.com")}
	private := urlPolicy{self: testSelf, allowedHosts: parseAllowedHosts("192.168.100.20, fd00::1")}

	cases := []struct {
		name    string
		url     string
		policy  urlPolicy
		wantErr string
	}{
		{"file scheme", "file:///etc/passwd", denyAll, "http or https"},
		{"gopher scheme", "gopher://example.com", denyAll, "http or https"},
		{"plain http public", "http://hooks.example.com/wa", urlPolicy{self: testSelf, allowedHosts: []string{"hooks.example.com"}}, "https for public hosts"},
		{"https public refused by default", "https://hooks.example.com/wa", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"host name refused by default", "https://n8n.home.example/hook", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"public ip refused by default", "https://93.184.216.34/hook", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"https public listed", "https://hooks.example.com/wa", urlPolicy{self: testSelf, allowedHosts: []string{"hooks.example.com"}}, ""},
		{"loopback refused when unset", "http://127.0.0.1:5678/webhook/wa", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"localhost refused when unset", "http://localhost:5678/hook", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"private network refused when unset", "http://192.168.100.20/hook", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"ipv6 unique local refused when unset", "http://[fd00::1]:8000/hook", denyAll, "WEBHOOK_ALLOWED_HOSTS"},
		{"local automation when listed", "http://127.0.0.1:5678/webhook/wa", local, ""},
		{"localhost when listed", "http://localhost:5678/hook", local, ""},
		{"private network when listed", "http://192.168.100.20/hook", private, ""},
		{"ipv6 unique local when listed", "http://[fd00::1]:8000/hook", private, ""},
		{"bridge itself", "http://127.0.0.1:8080/api/send", local, "bridge's own API"},
		{"bridge via localhost", "http://localhost:8080/api/send", local, "bridge's own API"},
		{"bridge via other loopback", "http://127.0.0.2:8080/api/send", urlPolicy{self: testSelf, allowedHosts: []string{"127.0.0.2"}}, "bridge's own API"},
		{"credentials", "https://user:pass@hooks.example.com/wa", allow, "user name or password"},
		{"no host", "https:///wa", denyAll, "must have a host"},
		{"metadata literal", "http://169.254.169.254/latest", urlPolicy{self: testSelf, allowedHosts: []string{"169.254.169.254"}}, "link-local"},
		{"too long", "https://hooks.example.com/" + strings.Repeat("a", maxWebhookURLLength), allow, "longer than"},
		{"empty", "", denyAll, "required"},
		{"allowlist hit", "https://n8n.example.org/hook", allow, ""},
		{"allowlist suffix", "https://a.hooks.example.com/hook", allow, ""},
		{"allowlist ip", "http://127.0.0.1:5678/hook", allow, ""},
		{"allowlist miss", "https://other.example.net/hook", allow, "WEBHOOK_ALLOWED_HOSTS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookURL(tc.url, tc.policy)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

func TestMaskURLAndLogTarget(t *testing.T) {
	if got := maskURL("https://ntfy.example.org/alerts?auth=abc123"); got != "https://ntfy.example.org/alerts?auth=***" {
		t.Errorf("maskURL = %q", got)
	}
	if got := maskURL("http://127.0.0.1:5678/webhook/wa"); got != "http://127.0.0.1:5678/webhook/wa" {
		t.Errorf("maskURL without query = %q", got)
	}
	got := logTarget("https://hooks.example.org/T000/B000/XXXX?token=abc")
	if got != "https://hooks.example.org" {
		t.Errorf("logTarget = %q", got)
	}
}

func TestForbiddenIP(t *testing.T) {
	cases := []struct {
		ip          string
		port        int
		allowPublic bool
		forbidden   bool
	}{
		{"169.254.169.254", 80, true, true},
		{"fe80::1", 80, true, true},
		{"0.0.0.0", 80, true, true},
		{"224.0.0.1", 80, true, true},
		{"255.255.255.255", 80, true, true},
		{"127.0.0.1", 8080, true, true}, // the bridge itself
		{"127.0.0.1", 5678, false, false},
		{"192.168.1.10", 8080, false, false},
		{"fd00::1", 443, false, false},
		{"93.184.216.34", 443, false, true}, // public, no WEBHOOK_ALLOWED_HOSTS
		{"93.184.216.34", 443, true, false}, // public, allowlist set
		{"::ffff:93.184.216.34", 443, false, true},
	}
	for _, tc := range cases {
		if got := forbiddenIP(net.ParseIP(tc.ip), tc.port, testSelf, tc.allowPublic); got != tc.forbidden {
			t.Errorf("forbiddenIP(%s:%d) = %v, want %v", tc.ip, tc.port, got, tc.forbidden)
		}
	}
}

func TestDialControlRefusesMetadataBeforeConnecting(t *testing.T) {
	client := newWebhookHTTPClient(2*time.Second, urlPolicy{self: testSelf, allowedHosts: []string{"169.254.169.254"}})
	start := time.Now()
	_, err := client.Post("http://169.254.169.254/latest/meta-data", "application/json", nil)
	if err == nil || !strings.Contains(err.Error(), "refused by the webhook URL policy") {
		t.Fatalf("err = %v, want a policy refusal", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("refusal took %v; it should happen before any network wait", time.Since(start))
	}
	_ = http.StatusOK
}

func TestDialControlRefusesEverythingWhenAllowlistUnset(t *testing.T) {
	client := newWebhookHTTPClient(2*time.Second, urlPolicy{self: testSelf})
	for _, target := range []string{
		"http://127.0.0.1:5678/hook",
		"http://192.168.1.10/hook",
		"https://93.184.216.34/hook",
	} {
		_, err := client.Post(target, "application/json", nil)
		if err == nil || !strings.Contains(err.Error(), "WEBHOOK_ALLOWED_HOSTS is unset") {
			t.Errorf("%s: err = %v, want unset-allowlist refusal", target, err)
		}
	}
}
