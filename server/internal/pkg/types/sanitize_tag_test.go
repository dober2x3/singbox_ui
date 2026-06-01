package types

import (
	"testing"
)

func TestSanitizeTag_ipv4(t *testing.T) {
	got := SanitizeTag("vmess", "1.2.3.4", 443)
	want := "vmess-1_2_3_4-443"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_ipv6(t *testing.T) {
	got := SanitizeTag("vless", "2001:db8::1", 8080)
	want := "vless-2001_db8__1-8080"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_specialChars(t *testing.T) {
	got := SanitizeTag("ss", "host-name.com", 8388)
	want := "ss-host_name_com-8388"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_emptyAddress(t *testing.T) {
	got := SanitizeTag("direct", "", 0)
	want := "direct--0"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_predefined(t *testing.T) {
	got := ResolveUserAgent("clash-verge")
	want := "clash-verge/v2.4.0"
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_custom(t *testing.T) {
	got := ResolveUserAgent("MyCustomUA/1.0")
	want := "MyCustomUA/1.0"
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_empty(t *testing.T) {
	got := ResolveUserAgent("")
	want := PredefinedUserAgents["default"]
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}
