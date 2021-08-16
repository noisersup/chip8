package main

import (
	"testing"
)

var ch *Chip8

func TestMain(m *testing.M) {
	var tab [4096]uint8
	ch = &Chip8{pc: 0, memory: tab}
	ch.Initialize(fontset)
	m.Run()
}

func TestFetchOpcode(t *testing.T) {
	expected := uint16(0xA2F0)

	c := Chip8{pc: 0}
	c.memory[c.pc] = 0xA2
	c.memory[c.pc+1] = 0xF0
	c.fetchOpcode()

	if c.opcode != expected {
		t.Errorf("Opcode different than expected %#04x (expected: %#04x)", c.opcode, expected)
	}
}

func Test1NNN(t *testing.T) {
	expected := uint16(0x024e)

	ch.opcode = 0x124e
	ch.decodeOpcode()
	if ch.pc != expected {
		t.Errorf("Opcode different than expected %#04x (expected: %#04x)", ch.opcode, expected)
	}
}

func Test6XNN(t *testing.T) {
	expected := uint8(0x01)

	ch.opcode = 0x6801
	ch.decodeOpcode()
	if ch.V[8] != expected {
		t.Errorf("Value different than expected %#04x (expected: %#04x)", ch.V[8], expected)
	}
}
