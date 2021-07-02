package main

import "log"

func main() {
	var tab [4096]uint8
	tab[0] = 233
	tab[1] = 112
	ch8 := Chip8{pc: 0, memory: tab}
	ch8.fetchOpcode()
}

type Chip8 struct {
	opcode uint16
	memory [4096]uint8

	//Cpu registers
	V [16]uint8

	//Index register
	I uint8

	//Program counter
	pc uint16

	//Graphics 64x32 resolution
	gfx [64 * 32]uint8

	// 60Hz timers (when set above zero they will count down to it)
	delayTimer uint8
	soundTimer uint8

	stack [16]uint8
	// Stack pointer
	sp uint8

	// Keypad
	key [16]uint8
}

func (ch8 *Chip8) Initialize(chip8Fontset []uint8) {
	ch8.pc = 0x200 // Program counter starts at 0x200
	ch8.opcode = 0 //Reset opcode
	ch8.I = 0      //Reset index register
	ch8.sp = 0     //Reset stack pointer

	/*TODO:
	- clear display
	- clear stack
	- clear registers V0-VF
	- clear memory
	*/

	for i := 0; i < 80; i++ {
		ch8.memory[i] = chip8Fontset[i]
	}
	// TODO:Reset timers
}

func (ch8 *Chip8) LoadProgram(buffer []uint8) {
	// Load all bytes from buffer to Chip8 memory from 0x200 (=512)
	for i, data := range buffer {
		ch8.memory[512+i] = data
	}
}

func (ch8 *Chip8) EmulateCycle() {
	ch8.fetchOpcode()
}

func (ch8 *Chip8) fetchOpcode() {
	/*
			memory[pc] = 11010001
			memory[pc+1] = 01010011

			11010001 <<8 (1101000100000000)

			1101000100000000 | 01010011 <- OR

			1101000100000000
		OR	0000000001010011
			----------------
			1101000101010011
	*/
	ch8.opcode = uint16(ch8.memory[ch8.pc])<<8 | uint16(ch8.memory[ch8.pc+1])
}

func (ch8 *Chip8) decodeOpcode() {
	switch ch8.opcode & 0xF000 { // Checks first byte
	case 0x0000:
		switch ch8.opcode & 0x000F { // Checks last byte
		case 0x0000: // 0x00E0: clears the screen
			//TODO
			break
		case 0x000E: // 0x00EE: Returns from subroutine
			//TODO
			break
		default:
			log.Printf("Unknown opcode [0x0000]: 0x%X\n", ch8.opcode)
		}
		break

	case 0x1000:
		break

	case 0x2000:
		break

	case 0x3000:
		break

	case 0x4000:
		break

	case 0x5000:
		break

	case 0x6000:
		break

	case 0x7000:
		break

	case 0x8000:
		break

	case 0x9000:
		break

	case 0xA000:
		break

	case 0xD000: // Draw a sprite
		break
	}

}
