package display

import "testing"

func TestStableMonitorIDPrefersDeviceID(t *testing.T) {
	got := StableMonitorID(`MONITOR\GSM5BEE\{4d36e96e-e325-11ce-bfc1-08002be10318}\0001`, "deadbeef")
	want := `MONITOR\GSM5BEE\{4d36e96e-e325-11ce-bfc1-08002be10318}\0001`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStableMonitorIDFallsBackToEDIDHash(t *testing.T) {
	got := StableMonitorID("", "aabbccdd")
	if got != "edid:aabbccdd" {
		t.Fatalf("got %q", got)
	}
}

func TestIsGDIDisplayName(t *testing.T) {
	if !IsGDIDisplayName(`\\.\DISPLAY1`) {
		t.Fatal("expected GDI name")
	}
	if IsGDIDisplayName(`MONITOR\GSM\1`) {
		t.Fatal("PnP path is not GDI")
	}
}
