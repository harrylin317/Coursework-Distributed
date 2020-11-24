package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

var (
	world       [][]byte
	aliveCells  []util.Cell
	turns       int
	imageHeight int
	imageWidth  int
)

// distributor divides the work between workers and interacts with other goroutines.
func main() {
	pAddr := flag.String("ip", ":8050", "port to listen on")
	flag.Parse()
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Register(&DistributorOperation{})
	rpc.Accept(listener)

}

type DistributorOperation struct{}

func (d *DistributorOperation) Calculate(req stubs.RequiredValue, res *stubs.Response) (err error) {
	world = calculateNextState(req.ImageHeight, req.ImageWidth, world)

	return
}

func (d *DistributorOperation) KeyPressed(req stubs.RequiredValue, res *stubs.World) (err error) {
	return
}
func (d *DistributorOperation) SendValues(req stubs.RequiredValue, res *stubs.Response) (err error) {
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	turns = req.Turns
	fmt.Println(world, imageHeight, imageWidth, turns)

	return
}
func (d *DistributorOperation) GetAliveCells(req stubs.RequiredValue, res *stubs.AliveCells) (err error) {
	res.AliveCells = calculateAliveCells(req.ImageHeight, req.ImageWidth, req.World)
	return
}
func (d *DistributorOperation) GenerateOutput(req stubs.RequiredValue, res *stubs.AliveCells) (err error) {
	return
}
func (d *DistributorOperation) Test(req stubs.Command, res *stubs.Response) (err error) {
	return
}

func mod(x, m int) int {
	return (x + m) % m
}

//make empty new world with given dimension
func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

//calculates surrounding neighbours
func calculateNeighbours(x, y, imageHeight, imageWidth int, world [][]byte) int {
	neighbours := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				if world[mod(y+i, imageHeight)][mod(x+j, imageWidth)] == alive {
					neighbours++
				}
			}
		}
	}
	return neighbours
}

//calculates next state for the world with the provided dimension, varys from workers
func calculateNextState(imageHeight, imageWidth int, world [][]byte) [][]byte {
	// height := endY - startY
	// width := endX - startX
	newWorld := makeWorld(imageHeight, imageWidth)

	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			neighbours := calculateNeighbours(x, y, imageHeight, imageWidth, world)
			//call CellFlipped event when a cell state is changed
			if world[y][x] == alive {
				if neighbours == 2 || neighbours == 3 {
					newWorld[y][x] = alive
				} else {
					newWorld[y][x] = dead
				}
			} else {
				if neighbours == 3 {
					newWorld[y][x] = alive
				} else {
					newWorld[y][x] = dead
				}
			}
		}
	}
	return newWorld
}

//calculate all alive cells and store into a slice
func calculateAliveCells(imageHeight, imageWidth int, world [][]byte) []util.Cell {
	aliveCells := []util.Cell{}
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageHeight; x++ {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}
