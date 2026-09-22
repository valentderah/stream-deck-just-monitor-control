package streamdeck

import "testing"

func TestShowAlertMessage(t *testing.T) {
	m := showAlertMessage("ctx-1")
	if m["event"] != "showAlert" || m["context"] != "ctx-1" {
		t.Fatalf("%v", m)
	}
}

func TestShowOkMessage(t *testing.T) {
	m := showOkMessage("c")
	if m["event"] != "showOk" {
		t.Fatalf("%v", m)
	}
}
