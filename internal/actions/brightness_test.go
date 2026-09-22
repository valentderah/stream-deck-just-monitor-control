package actions

import "testing"

func TestBrightnessStep(t *testing.T) {
	if got := applyStep(50, 10, 100); got != 60 {
		t.Fatal(got)
	}
	if got := applyStep(95, 10, 100); got != 100 {
		t.Fatal(got)
	}
	if got := applyStep(5, -10, 100); got != 0 {
		t.Fatal(got)
	}
}

func TestBrightnessToggle(t *testing.T) {
	if applyToggle(20, 20, 80) != 80 {
		t.Fatal()
	}
	if applyToggle(80, 20, 80) != 20 {
		t.Fatal()
	}
}
