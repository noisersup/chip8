package display

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
)

const (
	width  = 640
	height = 320

	vertexShaderSource = `
    #version 410
    in vec3 vp;
    void main() {
        gl_Position = vec4(vp, 1.0);
    }
` + "\x00"

	fragmentShaderSource = `
    #version 410
    out vec4 frag_colour;
    void main() {
        frag_colour = vec4(1, 1, 1, 1);
    }
` + "\x00"

	rows    = 32
	columns = 64
)

var (
	ratio  = float32(height) / float32(width)
	square = []float32{
		ratio * -0.5, 0.5, 0, //top
		ratio * -0.5, -0.5, 0, //left
		ratio * 0.5, -0.5, 0, //right

		ratio * -0.5, 0.5, 0, //top
		ratio * 0.5, 0.5, 0, //left
		ratio * 0.5, -0.5, 0, //right
	}
)

type Screen struct {
	window  *glfw.Window
	program uint32

	cells [][]*cell
}

func InitScreen() (*Screen, error) {
	window, err := initGlfw()
	if err != nil {
		return nil, err
	}

	program, err := initOpenGL()

	s := Screen{window, program, makeCells()}
	return &s, nil
}

func (s *Screen) Terminate() {
	glfw.Terminate()
}

func (s *Screen) ShouldClose() bool {
	return s.window.ShouldClose()
}

func (s *Screen) Draw(gfx []uint8) {
	draw(gfx, s.cells, s.window, s.program)
}

type cell struct {
	drawable uint32

	x int
	y int
}

func makeCells() [][]*cell {
	cells := make([][]*cell, columns, columns)
	for x := 0; x < columns; x++ {
		for y := 0; y < rows; y++ {
			c := newCell(x, y)
			cells[x] = append(cells[x], c)
		}
	}

	return cells
}

func newCell(x, y int) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		size = 1.0 / 64

		switch i % 3 {
		case 0:
			position = float32(x) * size
		case 1:
			size = size / ratio
			position = float32(y) * size
		default:
			continue
		}

		if points[i] < 0 {
			points[i] = (position * 2) - 1
		} else {
			points[i] = ((position + size) * 2) - 1
		}
	}

	return &cell{
		drawable: makeVao(points),

		x: x,
		y: y,
	}
}

func (c *cell) draw() {
	gl.BindVertexArray(c.drawable)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square)/3))
}

/*

	private display functions

*/

// initialize Window
func initGlfw() (*glfw.Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, err
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // OR 2
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, "Tutorial", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	return window, nil
}

// initialize OpenGL
func initOpenGL() (uint32, error) {
	if err := gl.Init(); err != nil {
		return 0, err
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)
	return prog, nil
}

// Creates a vertex array from points provided
func makeVao(points []float32) uint32 {
	var vbo uint32 //vertex buffer
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32 // vertex array
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("Failed to cimpile %v: %v", source, log)
	}
	return shader, nil
}

// draws to 2nd buffer and swaps them
func draw(gfx []uint8, cells [][]*cell, window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT) //removes all from second buffer
	gl.UseProgram(program)

	for x := range cells {
		for i, c := range cells[x] {
			if gfx[x*rows+i] != 0 {
				c.draw()
			}
		}
	}

	glfw.PollEvents() //checks for user input
	window.SwapBuffers()
}
