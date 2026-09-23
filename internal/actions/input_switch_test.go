package actions

import "testing"

func toggleDecision(cur uint32, readOK bool, portA, portB uint32, currentState int) (target uint32, nextState int) {
	if readOK && cur > 0 {
		if cur == portA {
			return portB, 1
		}
		if cur == portB {
			return portA, 0
		}
	}
	if currentState == 0 {
		return portB, 1
	}
	return portA, 0
}

func TestToggleDecision(t *testing.T) {
	const portA, portB uint32 = 0x0F, 0x11

	target, state := toggleDecision(portA, true, portA, portB, 0)
	if target != portB || state != 1 {
		t.Fatalf("on Port A: got target=%d state=%d", target, state)
	}

	target, state = toggleDecision(portB, true, portA, portB, 1)
	if target != portA || state != 0 {
		t.Fatalf("on Port B: got target=%d state=%d", target, state)
	}

	// GetVCP returned 0 / timeout — must flip by saved button state, not fall back to Port A
	target, state = toggleDecision(0, false, portA, portB, 0)
	if target != portB || state != 1 {
		t.Fatalf("read fail from state 0: got target=%d state=%d", target, state)
	}

	target, state = toggleDecision(0, true, portA, portB, 1)
	if target != portA || state != 0 {
		t.Fatalf("cur=0 from state 1: got target=%d state=%d", target, state)
	}

	// Third-party input — step by saved state
	target, state = toggleDecision(0x01, true, portA, portB, 0)
	if target != portB || state != 1 {
		t.Fatalf("other input from state 0: got target=%d state=%d", target, state)
	}
}
