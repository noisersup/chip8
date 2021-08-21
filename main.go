package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/noisersup/chip8/display"
)

var (
	square = []float32{
		-0.5, 0.5, 0, // top
		-0.5, -0.5, 0, // left
		0.5, -0.5, 0, // right

		-0.5, 0.5, 0, // top
		0.5, 0.5, 0, // left
		0.5, -0.5, 0, // right
	}
	fontset = []uint8{
		0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
		0x20, 0x60, 0x20, 0x20, 0x70, // 1
		0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
		0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
		0x90, 0x90, 0xF0, 0x10, 0x10, // 4
		0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
		0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
		0xF0, 0x10, 0x20, 0x40, 0x40, // 7
		0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
		0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
		0xF0, 0x90, 0xF0, 0x90, 0x90, // A
		0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
		0xF0, 0x80, 0x80, 0x80, 0xF0, // C
		0xE0, 0x90, 0x90, 0x90, 0xE0, // D
		0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
		0xF0, 0x80, 0xF0, 0x80, 0x81, // F
	}
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Specify path to ROM!")
	}
	filepath := os.Args[1]
	_, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Panic(err)
	}

	a := NewApp(filepath)

	if err := tea.NewProgram(a).Start(); err != nil {
		log.Fatal(err)
	}
}

type UpdateDebugger func()
type Chip8 struct {
	screen *display.Screen
	opcode uint16
	memory [4096]uint8

	//Cpu registers
	V [16]uint8

	//Index register
	I uint16

	//Program counter
	pc uint16

	//Graphics 64x32 resolution
	gfx []uint8

	// 60Hz timers (when set above zero they will count down to it)
	delayTimer uint8
	soundTimer uint8

	stack [16]uint16
	// Stack pointer
	sp uint16

	// Keypad
	keys [16]uint8

	debugMode bool
	stepChan  chan bool
	tick      int

	updDbg UpdateDebugger
}

func (ch8 *Chip8) Initialize(chip8Fontset []uint8) {
	ch8.tick = 0
	ch8.pc = 0x200 // Program counter starts at 0x200
	ch8.opcode = 0 //Reset opcode
	ch8.I = 0      //Reset index register
	ch8.sp = 0     //Reset stack pointer
	rand.Seed(time.Now().UTC().UnixNano())
	/*TODO:
	- clear display
	- clear stack
	- clear registers V0-VF
	- clear memory
	*/

	for i := 0; i < 64*32; i++ {
		ch8.gfx = append(ch8.gfx, 0)
	}

	for i := 0; i < 80; i++ {
		ch8.memory[i] = chip8Fontset[i]
	}
	// TODO:Reset timers
}

func (c *Chip8) LoadProg(fileName string) error {
	file, fileErr := os.OpenFile(fileName, os.O_RDONLY, 0777)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	fStat, fStatErr := file.Stat()
	if fStatErr != nil {
		return fStatErr
	}
	if int64(len(c.memory)-512) < fStat.Size() { // program is loaded at 0x200
		return fmt.Errorf("Program size bigger than memory")
	}

	buffer := make([]byte, fStat.Size())
	if _, readErr := file.Read(buffer); readErr != nil {
		return readErr
	}

	for i := 0; i < len(buffer); i++ {
		c.memory[i+512] = buffer[i]
	}

	return nil
}

func (ch8 *Chip8) LoadProgram(buffer []uint8) {
	// Load all bytes from buffer to Chip8 memory from 0x200 (=512)
	for i, data := range buffer {
		ch8.memory[512+i] = data
	}
}

var tick uint

func (ch8 *Chip8) EmulateCycle() {
	if ch8.debugMode {
		<-ch8.stepChan
	}
	tick++
	ch8.fetchOpcode()
	ch8.decodeOpcode()
	ch8.updDbg()
}

func showDiff(tab1, tab2 [4096]uint8) {
	fmt.Println("Memory differences:")
	for i, n := range tab1 {
		if n != tab2[i] {
			fmt.Printf("[%d]: %d->%d\n", i, n, tab2[i])
		}
	}
	fmt.Println()
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

func (ch8 *Chip8) Step() {
	if !ch8.debugMode {
		return
	}
	ch8.stepChan <- true
}

func (ch8 *Chip8) decodeOpcode() {
	switch ch8.opcode & 0xF000 { // Checks first byte
	case 0x0000:
		switch ch8.opcode & 0x000F { // Checks last byte
		case 0x0000: // 0x00E0: clears the screen
			//TODO
			break
		case 0x000E: // 0x00EE: Returns from subroutine
			ch8.pc = ch8.stack[ch8.sp]
			ch8.sp--
			break
		default:
			log.Printf("Unknown opcode [0x0000]: 0x%X\n", ch8.opcode)
		}
		break

	case 0x1000: // 0x1NNN: Jumps to address NNN
		ch8.pc = ch8.opcode & 0x0FFF
		break

	case 0x2000: // 0x2NNN: Calls subroutine at NNN
		ch8.sp++
		ch8.stack[ch8.sp] = ch8.pc
		ch8.pc = ch8.opcode & 0x0FFF
		break

	case 0x3000: // 0x3XNN: Skips the next instruction if VX == NN
		if ch8.V[ch8.opcode&0x0F00] == uint8(ch8.opcode&0x00FF) {
			ch8.pc += 2
		}
		break

	case 0x4000: // 0x4XNN: Skips the next instruction if VX != NN
		if ch8.V[ch8.opcode&0x0F00] != uint8(ch8.opcode&0x00FF) {
			ch8.pc += 2
		}
		break

	case 0x5000: // 0x5XY0: Skips the next instruction if VX == VY
		if ch8.V[ch8.opcode&0x0F00] == ch8.V[ch8.opcode&0x00F0] {
			ch8.pc += 2
		}
		break

	case 0x6000: // 0x6XNN: Loads NN into VX
		ch8.V[(ch8.opcode&0x0F00)>>8] = uint8(ch8.opcode & 0x00FF)
		ch8.pc += 2
		break

	case 0x7000: // 0x7XNN: VX = VX + NN
		ch8.V[ch8.opcode&0x0F00] = ch8.V[ch8.opcode&0x0F00] + uint8(ch8.opcode&0x00FF)
		ch8.pc += 2
		break

	case 0x8000:
		vx := ch8.opcode & 0x0F00
		vy := ch8.opcode & 0x00F0

		switch ch8.opcode & 0x000F {
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
			ch8.V[vx] = ch8.V[vx] >> ch8.V[vy]
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
			ch8.V[vx] = ch8.V[vx] << ch8.V[vy]
			break

		default:
			log.Printf("Unknown opcode [0x8000]: 0x%X\n", ch8.opcode)
		}
		ch8.pc += 2
		break

	case 0x9000: // 0x9XY0: Skips the instruction if VX != VY
		if ch8.V[ch8.opcode&0x0F00] != ch8.V[ch8.opcode&0x00F0] {
			ch8.pc += 2
		}
		ch8.pc += 2
		break

	case 0xA000: // 0xANNN: Sets the value of I to NNN
		ch8.I = ch8.opcode & 0x0FFF
		ch8.pc += 2
		break

	case 0xB000: // 0xBNNN: Jumps to the address NNN + V0
		ch8.pc = ch8.opcode&0x0FFF + uint16(ch8.V[0])
		break

	case 0xC000: // 0xCXNN: sets VX to AND operation with random number and NN
		ch8.V[ch8.opcode&0x0F00] = uint8(ch8.opcode&0x00FF) & uint8(rand.Intn(256))
		ch8.pc += 2
		break

	case 0xD000: // 0xDXYN: Draw a sprite
		x := uint16(ch8.V[(ch8.opcode&0x0F00)>>8])
		y := uint16(ch8.V[(ch8.opcode&0x00F0)>>4])
		height := ch8.opcode & 0x000F
		var px uint8

		ch8.V[0xF] = 0
		for ySprite := uint16(0); ySprite < height; ySprite++ {

			px = ch8.memory[ch8.I+ySprite]
			for xSprite := uint16(0); xSprite < 8; xSprite++ {
				if (px & (0x80 >> xSprite)) != 0 {
					if ch8.gfx[x+xSprite+((y+ySprite)*60)] == 1 {
						ch8.V[0xF] = 1
					}
					ch8.gfx[x+xSprite+((y+ySprite)*60)] ^= 1
				}
			}
		}
		if !ch8.screen.ShouldClose() {
			ch8.screen.Draw(ch8.gfx)
		}
		ch8.pc += 2
		break
	case 0xE000:
		ch8.pc += 2
		switch ch8.opcode & 0x000F {
		case 0x000E: // 0xEX9E: Skips next instruction if key stored in VX is pressed
			if ch8.keys[ch8.V[ch8.opcode&0x0F00]] == 1 {
				ch8.pc += 2
			}
			break

		case 0x0001: // 0xEXA1: Skips next instruction if key stored in VX is not pressed
			if ch8.keys[ch8.V[ch8.opcode&0x0F00]] == 0 {
				ch8.pc += 2
			}
			break
		}
		break
	case 0xF000:
		vx := ch8.opcode & 0x0F00

		switch ch8.opcode & 0x00FF {
		case 0x0007: // 0xFX07: Read delay timer into VX
			ch8.V[vx] = ch8.delayTimer
			break
		case 0x000A: // 0xFX0A: Wait for a key press and store into VX
			//TODO
			break

		case 0x0015: // 0xFX15: Load VX to delay timer
			ch8.delayTimer = ch8.V[vx]
			break

		case 0x0018: // 0xFX18: Load VX to sound timer
			ch8.soundTimer = ch8.V[vx]
			break

		case 0x001E: // 0xFX1E: Adds VX to I
			ch8.I += uint16(ch8.V[vx])
			break

		case 0x0029: // 0xFX29: Set I to the location of sprite in VX
			ch8.I = uint16(ch8.V[vx] * 0x05)
			break

		case 0x0033: // 0xFX33: Stores BCD representation of VX
			hundreds := vx / 100
			tens := (vx - hundreds*100) / 10
			ones := vx - hundreds*100 - tens*10
			ch8.memory[ch8.I] = uint8(hundreds)
			ch8.memory[ch8.I+1] = uint8(tens)
			ch8.memory[ch8.I+2] = uint8(ones)

		case 0x0055: // 0xFX55: Stores V0 to VX in memory starting at I without modifing I.
			for i, register := range ch8.V {
				ch8.memory[ch8.I+uint16(i)] = register
			}
			break
		case 0x0065: // 0xFX65: Loads to V0 to VX from memory starting at I without modifing I.
			for i := range ch8.V {
				ch8.V[i] = ch8.memory[ch8.I+uint16(i)]
			}
			break
		}
		ch8.pc += 2
		ch8.pc += 2
		break
	}
}

type app struct {
	ch8         *Chip8
	refreshChan chan bool
}

func NewApp(filepath string) *app {
	var tab [4096]uint8

	runtime.LockOSThread()
	screen := display.InitScreen(640, 320, 64, 32, "aaa")

	stepChan := make(chan bool)
	refreshChan := make(chan bool)

	ch8 := Chip8{pc: 0, memory: tab, debugMode: true, stepChan: stepChan}
	a := app{ch8: &ch8, refreshChan: refreshChan}
	ch8.updDbg = a.refresh

	ch8.Initialize(fontset)
	//ch8.LoadProgram(rom)
	ch8.LoadProg(filepath)

	var gfx []uint8
	for i := 0; i < 64*32; i++ {
		gfx = append(gfx, 0)
	}

	screen.Draw([]uint8(gfx))
	go func() {
		for !screen.ShouldClose() {
			ch8.EmulateCycle()
			time.Sleep(500 * time.Millisecond)
		}
	}()
	return &a
}

type refreshMsg bool

func (a *app) refresh() {
	a.refreshChan <- true
}
func (a *app) waitForRefresh() tea.Cmd {
	return func() tea.Msg { return refreshMsg(<-a.refreshChan) }
}
func (a *app) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, a.waitForRefresh())
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "space":
			a.ch8.debugMode = !a.ch8.debugMode
		case "n":
			a.ch8.Step()
		}

	case refreshMsg:
		return a, a.waitForRefresh()
	}
	return a, nil
}

func (a *app) View() string {
	str := "DEBUGGER\n\n"
	str += fmt.Sprintf("OPCODE: %#04x\n", a.ch8.opcode)
	str += fmt.Sprintf("I: %#04x\n", a.ch8.I)
	str += fmt.Sprintf("PC: %#04x\n", a.ch8.pc)
	str += fmt.Sprintf("TICK: %d\n\n", tick)
	str += "STACK"

	for i := uint16(0); i < 16; i++ {
		str += fmt.Sprintf("\nS[%X]", i)
		if a.ch8.sp == i {
			str += "<-"
		} else {
			str += "  "
		}

		str += fmt.Sprintf(" = %#04x", a.ch8.stack[i])
		//Registers
		str += fmt.Sprintf("	V[%X]  = %#04x", i, a.ch8.V[i])
		//Memory
		currPc := a.ch8.pc + (i * 2)
		currOpcode := uint16(a.ch8.memory[currPc])<<8 | uint16(a.ch8.memory[currPc+1])
		str += fmt.Sprintf(" | [%04x]: %#04x", currPc, currOpcode)
	}

	return str
}
