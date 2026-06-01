package types

import "testing"

// TestIsProxyOutboundType_known verifies known proxy types return true.
func TestIsProxyOutboundType_known(t *testing.T) {
	if !IsProxyOutboundType("vless") {
		t.Error("IsProxyOutboundType('vless') = false, want true")
	}
}

// TestIsProxyOutboundType_unknown verifies unknown proxy types return false.
func TestIsProxyOutboundType_unknown(t *testing.T) {
	if IsProxyOutboundType("freedom") {
		t.Error("IsProxyOutboundType('freedom') = true, want false")
	}
}
