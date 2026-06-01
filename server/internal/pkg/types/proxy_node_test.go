package types

import "testing"

func TestIsProxyOutboundType_known(t *testing.T) {
	if !IsProxyOutboundType("vless") {
		t.Error("IsProxyOutboundType('vless') = false, want true")
	}
}

func TestIsProxyOutboundType_unknown(t *testing.T) {
	if IsProxyOutboundType("freedom") {
		t.Error("IsProxyOutboundType('freedom') = true, want false")
	}
}
