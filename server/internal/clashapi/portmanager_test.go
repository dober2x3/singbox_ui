package clashapi

import (
	"path/filepath"
	"testing"
)

func TestPortManagerAssign(t *testing.T) {
	pm := NewPortManager(9090)
	p1 := pm.Assign("default")
	p2 := pm.Assign("office")
	p3 := pm.Assign("home")

	if p1 != 9090 {
		t.Fatalf("expected 9090, got %d", p1)
	}
	if p2 != 9091 {
		t.Fatalf("expected 9091, got %d", p2)
	}
	if p3 != 9092 {
		t.Fatalf("expected 9092, got %d", p3)
	}
}

func TestPortManagerReleaseAndReassign(t *testing.T) {
	pm := NewPortManager(9090)
	_ = pm.Assign("default")  // 9090
	_ = pm.Assign("office")   // 9091
	pm.Release("office")
	p := pm.Assign("home") // должно переиспользовать 9091
	if p != 9091 {
		t.Fatalf("expected 9091 (reuse), got %d", p)
	}
}

func TestPortManagerGet(t *testing.T) {
	pm := NewPortManager(9090)
	pm.Assign("default")
	port, ok := pm.Get("default")
	if !ok || port != 9090 {
		t.Fatalf("expected 9090, got %d (ok=%v)", port, ok)
	}
	_, ok = pm.Get("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent instance")
	}
}

func TestPortManagerPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ports.json")

	pm1 := NewPortManager(9090)
	pm1.Assign("default")
	pm1.Assign("office")
	if err := pm1.Save(path); err != nil {
		t.Fatal(err)
	}

	pm2 := NewPortManager(9090)
	if err := pm2.Load(path); err != nil {
		t.Fatal(err)
	}
	port, ok := pm2.Get("default")
	if !ok || port != 9090 {
		t.Fatalf("default: expected 9090, got %d", port)
	}
	port, ok = pm2.Get("office")
	if !ok || port != 9091 {
		t.Fatalf("office: expected 9091, got %d", port)
	}
}
