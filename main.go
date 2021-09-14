package main

import (
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/noisersup/chip8/chip8"
	"github.com/noisersup/chip8/display"
	"github.com/noisersup/chip8/models"
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
		//log.Fatal("Specify path to ROM!")
		os.Args = append(os.Args, "Space Invaders [David Winter].ch8")
	}

	switch os.Args[1] {
	case "test": // Display screen test
		var gfx []uint8
		gfx = make([]uint8, 64*32, 64*32)

		screen, err := display.InitScreen()
		if err != nil {
			panic(err)
		}
		for !screen.ShouldClose() {
			for i := 0; i < 64*32; i++ {
				gfx[i] = 1
				screen.Draw(gfx)
			}
		}
		break

	case "debug": // Emulate without cli
		log.Print("debug")
		screen, err := display.InitScreen()
		if err != nil {
			panic(err)
		}
		ch8 := chip8.NewChip8(screen)
		ch8.Initialize(fontset)
		ch8.LoadProgram("Space Invaders [David Winter].ch8")
		//ch8.LoadProgram("test.ch8")
		ch8.UpdDbg = func() {

		}

		/*go func() { // runs cpu
			for {
				ch8.Step()
			}
		}()*/
		ch8.DebugMode = false

		for !screen.ShouldClose() {
			ch8.EmulateCycle()
			//time.Sleep(500 * time.Millisecond)
		}

		break
	default: // Emulate with cli
		filepath := os.Args[1]
		a := NewApp(filepath)

		if err := tea.NewProgram(a).Start(); err != nil {
			log.Fatal(err)
		}
	}
}

type app struct {
	ch8         *chip8.Chip8
	refreshChan chan bool
}

func NewApp(filepath string) *app {
	screen, err := display.InitScreen()
	if err != nil {
		panic(err)
	}

	refreshChan := make(chan bool)

	ch8 := chip8.NewChip8(screen)

	a := app{ch8: ch8, refreshChan: refreshChan}
	ch8.UpdDbg = a.refresh

	ch8.Initialize(fontset)
	ch8.LoadProgram(filepath)

	go func() {
		for !screen.ShouldClose() {
			ch8.EmulateCycle()
			time.Sleep(500 * time.Millisecond)
		}
	}()
	return &a
}

func (a *app) refresh() {
	a.refreshChan <- true
}
func (a *app) waitForRefresh() tea.Cmd {
	return func() tea.Msg { return models.RefreshMsg(<-a.refreshChan) }
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
		case " ":
			a.ch8.DebugMode = !a.ch8.DebugMode
		case "n":
			a.ch8.Step()
		}

	case models.RefreshMsg:
		return a, a.waitForRefresh()
	}
	return a, nil
}

func (a *app) View() string {
	str := "DEBUGGER\n\n"
	str += fmt.Sprintf("OPCODE: %#04x\n", a.ch8.Opcode)
	str += fmt.Sprintf("I: %#04x\n", a.ch8.I)
	str += fmt.Sprintf("PC: %#04x\n", a.ch8.Pc)
	str += fmt.Sprintf("TICK: %d\n\n", a.ch8.Tick)
	str += "STACK"

	for i := uint16(0); i < 16; i++ {
		str += fmt.Sprintf("\nS[%X]", i)
		if a.ch8.Sp == i {
			str += "<-"
		} else {
			str += "  "
		}

		str += fmt.Sprintf(" = %#04x", a.ch8.Stack[i])
		//Registers
		str += fmt.Sprintf("	V[%X]  = %#04x", i, a.ch8.V[i])
		//Memory
		currPc := a.ch8.Pc + (i * 2)
		currOpcode := uint16(a.ch8.Memory[currPc])<<8 | uint16(a.ch8.Memory[currPc+1])
		str += fmt.Sprintf(" | [%04x]: %#04x", currPc, currOpcode)
	}

	return str
}
