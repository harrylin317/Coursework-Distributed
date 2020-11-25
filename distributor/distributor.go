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
	world                                     [][]byte
	turn, imageHeight, imageWidth, totalTurns int
	unblock                                   = make(chan bool)
	start                                     = make(chan bool)
	getTurnSignal                             = make(chan bool)
	getAliveCellsSignal                       = make(chan bool)
	turnChan                                  = make(chan int)
	aliveCellsChan                            = make(chan []util.Cell)
	turnCompleted                             = false
	aliveCells                                []util.Cell
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

func (d *DistributorOperation) Test(req stubs.Request, res *stubs.Turn) (err error) {
	i := 0
	for i = 0; i < 10; i++ {

		res.Turn = i
		fmt.Println("finished turn", i)

	}
	res.Turn = i
	return
}
func (d *DistributorOperation) StartTicker(req stubs.Request, res *stubs.Response) (err error) {

}

func (d *DistributorOperation) ExecuteAllTurns(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Start execution ", turn)
	fmt.Println("Total turn ", totalTurns)

	for turn = 0; turn < totalTurns; turn++ {
		fmt.Println("execurting turn ", turn)
		world = calculateNextState(imageHeight, imageWidth, world)
		aliveCells = calculateAliveCells(imageHeight, imageWidth, world)
		turnCompleted = true
		<-unblock
		turnChan <- turn + 1

	}

	return
}

func (d *DistributorOperation) GetWorld(req stubs.Request, res *stubs.World) (err error) {
	// for y := 0; y < imageHeight; y++ {
	// 	fmt.Println(world[y])
	// }
	res.World = world
	return
}
func (d *DistributorOperation) GetCurrentTurn(req stubs.Request, res *stubs.Turn) (err error) {
	unblock <- true
	getTurn := <-turnChan
	res.Turn = getTurn

	return
}

// func (d *DistributorOperation) KeyPressed(req stubs.Key, res *stubs.Response) (err error) {
// 	key := req.Key
// 	switch key {
// 	case 'q':
// 		exitChan <- true
// 	case 'p':
// 		if pause {
// 			pause = false
// 			fmt.Println("Continuing")
// 			pauseChan <- pause
// 		} else {
// 			pause = true
// 			pauseChan <- pause
// 		}

// 	}
// 	return
// }
func (d *DistributorOperation) InitializeValues(req stubs.RequiredValue, res *stubs.Response) (err error) {
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	totalTurns = req.Turns
	fmt.Println(imageHeight, imageWidth, totalTurns)
	aliveCells = calculateAliveCells(imageHeight, imageWidth, world)

	return
}
func (d *DistributorOperation) GetAliveCells(req stubs.Request, res *stubs.AliveCells) (err error) {
	res.AliveCells = aliveCells
	return
}
func (d *DistributorOperation) GenerateOutput(req stubs.RequiredValue, res *stubs.AliveCells) (err error) {
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
