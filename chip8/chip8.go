package chip8

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/noisersup/chip8/display"
	"github.com/noisersup/chip8/models"
)

type Chip8 struct {
	screen *display.Screen
	Opcode uint16
	Memory [4096]uint8

	//Cpu registers
	V [16]uint8

	//Index register
	I uint16

	//Program counter
	Pc uint16

	//Graphics 64x32 resolution
	Gfx []uint8

	// 60Hz timers (when set above zero they will count down to it)
	DelayTimer uint8
	SoundTimer uint8

	Stack [16]uint16
	// Stack pointer
	Sp uint16

	// Keypad
	keys [16]uint8

	DebugMode bool
	stepChan  chan bool
	Tick      int

	display chan<- []uint8
	UpdDbg  models.UpdateDebugger
}

func NewChip8(screen *display.Screen, gfx chan []uint8) *Chip8 {
	stepCh := make(chan bool)
	c := Chip8{screen: screen, Pc: 0, Memory: [4096]uint8{}, DebugMode: true, stepChan: stepCh, display: gfx}
	return &c
}

func (ch8 *Chip8) Initialize(chip8Fontset []uint8) {
	ch8.Tick = 0
	ch8.Pc = 0x200 // Program counter starts at 0x200
	ch8.Opcode = 0 //Reset opcode
	ch8.I = 0      //Reset index register
	ch8.Sp = 0     //Reset stack pointer
	rand.Seed(time.Now().UTC().UnixNano())
	/*TODO:
	- clear display
	- clear stack
	- clear registers V0-VF
	- clear memory
	*/

	ch8.Gfx = make([]uint8, 64*32, 64*32)
	ch8.display <- ch8.Gfx

	for i := 0; i < 80; i++ {
		ch8.Memory[i] = chip8Fontset[i]
	}
	// TODO:Reset timers
}

func (c *Chip8) LoadProgram(fileName string) error {
	file, fileErr := os.OpenFile(fileName, os.O_RDONLY, 0777)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	fStat, fStatErr := file.Stat()
	if fStatErr != nil {
		return fStatErr
	}
	if int64(len(c.Memory)-512) < fStat.Size() { // program is loaded at 0x200
		return fmt.Errorf("Program size bigger than memory")
	}

	buffer := make([]byte, fStat.Size())
	if _, readErr := file.Read(buffer); readErr != nil {
		return readErr
	}

	for i := 0; i < len(buffer); i++ {
		c.Memory[i+512] = buffer[i]
	}

	return nil
}

func (ch8 *Chip8) EmulateCycle() {
	if ch8.DebugMode {
		<-ch8.stepChan
	}
	if ch8.DelayTimer > 0 {
		for {
			time.Sleep(10 * time.Millisecond)
			ch8.DelayTimer--
			if ch8.DelayTimer <= 0 {
				break
			}
		}
	}
	ch8.Tick++
	ch8.fetchOpcode()
	ch8.decodeOpcode()
	ch8.UpdDbg()
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
	ch8.Opcode = uint16(ch8.Memory[ch8.Pc])<<8 | uint16(ch8.Memory[ch8.Pc+1])
	//log.Printf("0x%04x", ch8.Opcode)
}

func (ch8 *Chip8) Step() {
	if !ch8.DebugMode {
		return
	}
	ch8.stepChan <- true
}

func (ch8 *Chip8) ToggleDebug() {
	ch8.DebugMode = !ch8.DebugMode
	if !ch8.DebugMode {
		ch8.stepChan <- true
	}
}

func (ch8 *Chip8) decodeOpcode() {
	switch ch8.Opcode & 0xF000 { // Checks first byte
	case 0x0000:
		switch ch8.Opcode & 0x000F { // Checks last byte
		case 0x0000: // 0x00E0: clears the screen
			//TODO
			break
		case 0x000E: // 0x00EE: Returns from subroutine
			ch8.Pc = ch8.Stack[ch8.Sp]
			ch8.Sp--
			ch8.Pc += 2
			break
		default:
			log.Printf("Unknown opcode [0x0000]: 0x%X\n", ch8.Opcode)
		}
		break

	case 0x1000: // 0x1NNN: Jumps to address NNN
		ch8.Pc = ch8.Opcode & 0x0FFF
		break

	case 0x2000: // 0x2NNN: Calls subroutine at NNN
		ch8.Sp++
		ch8.Stack[ch8.Sp] = ch8.Pc
		ch8.Pc = ch8.Opcode & 0x0FFF
		break

	case 0x3000: // 0x3XNN: Skips the next instruction if VX == NN
		ch8.Pc += 2
		if ch8.V[ch8.Opcode&0x0F00>>8] == uint8(ch8.Opcode&0x00FF) {
			ch8.Pc += 2
		}
		break

	case 0x4000: // 0x4XNN: Skips the next instruction if VX != NN
		ch8.Pc += 2
		if ch8.V[ch8.Opcode&0x0F00>>8] != uint8(ch8.Opcode&0x00FF) {
			ch8.Pc += 2
		}
		break

	case 0x5000: // 0x5XY0: Skips the next instruction if VX == VY
		ch8.Pc += 2
		if ch8.V[ch8.Opcode&0x0F00>>8] == ch8.V[ch8.Opcode&0x00F0>>4] {
			ch8.Pc += 2
		}
		break

	case 0x6000: // 0x6XNN: Loads NN into VX
		ch8.V[(ch8.Opcode&0x0F00)>>8] = uint8(ch8.Opcode & 0x00FF)
		ch8.Pc += 2
		break

	case 0x7000: // 0x7XNN: VX = VX + NN
		ch8.V[(ch8.Opcode&0x0F00)>>8] = ch8.V[(ch8.Opcode&0x0F00)>>8] + uint8(ch8.Opcode&0x00FF)
		ch8.Pc += 2
		break

	case 0x8000:
		vx := ch8.Opcode & 0x0F00 >> 8
		vy := ch8.Opcode & 0x00F0 >> 4

		switch ch8.Opcode & 0x000F {
		case 0x0000: // 0x8XY0: Copy the value of VY to VX
			ch8.V[vx] = ch8.V[vy]
			break

		case 0x0001: // 0x8XY1: VX = VX OR VY
			ch8.V[vx] = ch8.V[vx] | ch8.V[vy]
			break

		case 0x0002: // 0x8XY2: VX = VX AND VY
			ch8.V[vx] = ch8.V[vx] & ch8.V[vy]
			break

		case 0x0003: // 0x8XY3: VX = VX XOR VY
			ch8.V[vx] = ch8.V[vx] ^ ch8.V[vy]
			break

		case 0x0004: // 0x8XY4: VX = VX ADD VY
			sum := ch8.V[vx] + ch8.V[vy]
			if sum > 0xFF {
				ch8.V[0xF] = 1
			} else {
				ch8.V[0xF] = 0
			}

			ch8.V[vx] = sum
			break

		case 0x0005: // 0x8XY5: VX = VX SUB VY
			if ch8.V[vx] > ch8.V[vy] {
				ch8.V[0xF] = 1
			} else {
				ch8.V[0xF] = 0
			}
			ch8.V[vx] = ch8.V[vx] - ch8.V[vy]
			break

		case 0x0006: // 0x8XY6: VX = VX >> VY
			ch8.V[0xF] = ch8.V[vy] & 1
			ch8.V[vx] = ch8.V[vy]
			ch8.V[vx] >>= 1
			break

		case 0x0007: // 0x8XY7: VX = VY SUB VX
			if ch8.V[vy] > ch8.V[vx] {
				ch8.V[0xF] = 1
			} else {
				ch8.V[0xF] = 0
			}
			ch8.V[vx] = ch8.V[vy] - ch8.V[vx]
			break

		case 0x000E: // 0x8XYE: VX = VX << VY
			ch8.V[0xF] = ch8.V[vy] & 1
			ch8.V[vx] = ch8.V[vy]
			ch8.V[vx] <<= 1
			break

		default:
			log.Printf("Unknown opcode [0x8000]: 0x%X\n", ch8.Opcode)
		}
		ch8.Pc += 2
		break

	case 0x9000: // 0x9XY0: Skips the instruction if VX != VY
		if ch8.V[ch8.Opcode&0x0F00>>8] != ch8.V[ch8.Opcode&0x00F0>>4] {
			ch8.Pc += 2
		}
		ch8.Pc += 2
		break

	case 0xA000: // 0xANNN: Sets the value of I to NNN
		ch8.I = ch8.Opcode & 0x0FFF
		ch8.Pc += 2
		break

	case 0xB000: // 0xBNNN: Jumps to the address NNN + V0
		ch8.Pc = ch8.Opcode&0x0FFF + uint16(ch8.V[0])
		break

	case 0xC000: // 0xCXNN: sets VX to AND operation with random number and NN
		ch8.V[ch8.Opcode&0x0F00>>8] = uint8(ch8.Opcode&0x00FF) & uint8(rand.Intn(256))
		ch8.Pc += 2
		break

	case 0xD000: // 0xDXYN: Draw a sprite
		x := uint16(ch8.V[(ch8.Opcode&0x0F00)>>8])
		y := uint16(ch8.V[(ch8.Opcode&0x00F0)>>4])
		height := uint16(ch8.Opcode & 0x000F)
		var px uint16

		ch8.V[0xF] = 0 // reset VF reg

		for row := uint16(0); row < height; row++ {
			px = uint16(ch8.Memory[ch8.I+row])

			for col := uint16(0); col < 8; col++ {
				if (px & (0x80 >> col)) != 0 {
					if ch8.Gfx[(x+col+((y+row)*64))] == 1 {
						ch8.V[0xF] = 1
					}
					ch8.Gfx[(x + col + ((y + row) * 64))] ^= 1
				}
			}
		}

		ch8.display <- ch8.Gfx
		ch8.Pc += 2
		break
	case 0xE000:
		ch8.Pc += 2
		switch ch8.Opcode & 0x000F {
		case 0x000E: // 0xEX9E: Skips next instruction if key stored in VX is pressed
			if ch8.keys[ch8.V[ch8.Opcode&0x0F00>>8]] == 1 {
				ch8.Pc += 2
			}
			break

		case 0x0001: // 0xEXA1: Skips next instruction if key stored in VX is not pressed
			if ch8.keys[ch8.V[ch8.Opcode&0x0F00>>8]] == 0 {
				ch8.Pc += 2
			}
			break
		}
		break
	case 0xF000:
		vx := (ch8.Opcode & 0x0F00) >> 8

		switch ch8.Opcode & 0x00FF {
		case 0x0007: // 0xFX07: Read delay timer into VX
			ch8.V[vx] = ch8.DelayTimer
		case 0x000A: // 0xFX0A: Wait for a key press and store into VX
			//TODO

		case 0x0015: // 0xFX15: Load VX to delay timer
			ch8.DelayTimer = ch8.V[vx]

		case 0x0018: // 0xFX18: Load VX to sound timer
			ch8.SoundTimer = ch8.V[vx]

		case 0x001E: // 0xFX1E: Adds VX to I
			ch8.I += uint16(ch8.V[vx])

		case 0x0029: // 0xFX29: Set I to the location of sprite in VX
			ch8.I = uint16(ch8.V[vx] * 0x05)

		case 0x0033: // 0xFX33: Stores BCD representation of VX
			hundreds := ch8.V[vx] / 100
			tens := (ch8.V[vx] - uint8(hundreds)*100) / 10
			ones := (ch8.V[vx] - (uint8(hundreds)*100 + tens*10))

			ch8.Memory[ch8.I] = uint8(hundreds)
			ch8.Memory[ch8.I+1] = uint8(tens)
			ch8.Memory[ch8.I+2] = uint8(ones)

		case 0x0055: // 0xFX55: Stores V0 to VX in memory starting at I without modifing I.
			for i := uint16(0); i <= ch8.Opcode&0x0F00>>8; i++ {
				ch8.Memory[ch8.I+uint16(i)] = ch8.V[i]
			}
		case 0x0065: // 0xFX65: Loads to V0 to VX from memory starting at I without modifing I.
			for i := uint16(0); i <= ch8.Opcode&0x0F00>>8; i++ {
				ch8.V[i] = ch8.Memory[ch8.I+uint16(i)]
			}
		}
		ch8.Pc += 2
		break
	}
}
