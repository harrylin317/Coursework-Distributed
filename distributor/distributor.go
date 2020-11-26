package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

var (
	world                                     [][]byte
	turn, imageHeight, imageWidth, totalTurns int
	unblock                                   = make(chan bool)
	turnChan                                  = make(chan int)
	aliveCellsChan                            = make(chan []util.Cell)
	cellFlippedChan                           = make(chan []util.Cell)
	pauseChan                                 = make(chan bool)
	haltChan                                  = make(chan bool)
	aliveCells, cellFlipped                   []util.Cell
	filename                                  string
	connectedToController                     = false
	connectionChan                            = make(chan bool)
	initialized                               = false
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

func (d *DistributorOperation) ExecuteAllTurns(req stubs.Request, res *stubs.Response) (err error) {
	for turn = 0; turn < totalTurns; turn++ {
		select {
		case <-pauseChan:
			fmt.Println("paused", turn)
			select {
			case <-pauseChan:
				fmt.Println("unpaused", turn)
				turn--
				break
			case connectedToController = <-connectionChan:
				turn--
				break
			}
		case connectedToController = <-connectionChan:
			turn--
			break
		default:
			cellFlipped = []util.Cell{}
			world = calculateNextState(imageHeight, imageWidth, world)
			aliveCells = calculateAliveCells(imageHeight, imageWidth, world)
			if connectedToController {
				<-unblock
				turnChan <- turn + 1
				aliveCellsChan <- aliveCells
				cellFlippedChan <- cellFlipped
			}
		}
	}
	initialized = false
	return
}

func (d *DistributorOperation) GetWorld(req stubs.Request, res *stubs.World) (err error) {
	res.World = world
	return
}
func (d *DistributorOperation) GetFilename(req stubs.Request, res *stubs.Filename) (err error) {
	res.Filename = filename
	return
}

func (d *DistributorOperation) GetCurrentState(req stubs.Request, res *stubs.State) (err error) {
	unblock <- true
	getTurn := <-turnChan
	getAliveCells := <-aliveCellsChan
	getCellFlipped := <-cellFlippedChan
	res.Turn = getTurn
	res.AliveCells = getAliveCells
	res.CellFlipped = getCellFlipped
	return
}
func (d *DistributorOperation) CheckIfInitialized(req stubs.Request, res *stubs.Initialized) (err error) {
	res.Initialized = initialized
	if !connectedToController {
		connectedToController = true
	}
	return
}
func (d *DistributorOperation) KeyPressed(req stubs.Key, res *stubs.Response) (err error) {
	key := req.Key
	switch key {
	case 's':
		res.Message = "Output"
	case 'q':
		connectionChan <- false
		filename = strconv.Itoa(imageWidth) + "x" + strconv.Itoa(imageHeight)
		res.Message = "Exit"
	case 'p':
		pauseChan <- true
		res.Message = "Pause"
	}
	return
}
func (d *DistributorOperation) InitializeValues(req stubs.RequiredValue, res *stubs.State) (err error) {
	initialized = true
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	totalTurns = req.Turns
	aliveCells = calculateAliveCells(imageHeight, imageWidth, world)
	res.Turn = turn
	res.AliveCells = aliveCells
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
					cellFlipped = append(cellFlipped, util.Cell{X: x, Y: y})
				}
			} else {
				if neighbours == 3 {
					newWorld[y][x] = alive
					cellFlipped = append(cellFlipped, util.Cell{X: x, Y: y})

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
