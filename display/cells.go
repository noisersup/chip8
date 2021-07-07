package display

import "github.com/go-gl/gl/v4.6-core/gl"

type cell struct {
	drawable uint32

	x int
	y int
}

func newCell(x, y, columns, rows int) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(columns)
			position = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
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
		drawable: MakeVao(points),
		x:        x,
		y:        y,
	}
}

func (c *cell) draw() {
	gl.BindVertexArray(c.drawable)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square)/3))
}

func MakeCells(rows, columns int) [][]*cell {
	cells := make([][]*cell, rows, rows)

	for x := 0; x < rows; x++ {
		for y := 0; y < columns; y++ {
			c := newCell(x, y, columns, rows)
			cells[x] = append(cells[x], c)
		}
	}
	return cells
}
