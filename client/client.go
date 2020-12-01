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
	nextNeighbourAddr          string
	previousNeighbourAddr      string
	world                      [][]byte
	startY, endY, startX, endX int
	aliveCells, cellFlipped    []util.Cell
)

func mod(x, m int) int {
	return (x + m) % m
}
func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}
func calculateAliveCells() []util.Cell {
	aliveCells := []util.Cell{}
	for y := 0; y < endY; y++ {
		for x := 0; x < endX; x++ {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}
func calculateNeighbours(x, y, imageHeight, imageWidth int) int {
	neighbours := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				if y+i == imageHeight {
					sendCoordinate := stubs.Coordinate{Y: y + i, X: x + j}
					isAlive := new(stubs.IsAlive)
					client, err := rpc.Dial("tcp", previousNeighbourAddr)
					if err != nil {
						fmt.Println("Error")
						fmt.Println(err)
					}
					client.Call(stubs.GetEdgeValue, sendCoordinate, isAlive)
					if isAlive.Alive {
						neighbours++
					}
				} else if y+i < 0 {
					sendCoordinate := stubs.Coordinate{Y: y + i, X: x + j}
					isAlive := new(stubs.IsAlive)
					client, err := rpc.Dial("tcp", nextNeighbourAddr)
					if err != nil {
						fmt.Println("Error")
						fmt.Println(err)
					}
					client.Call(stubs.GetEdgeValue, sendCoordinate, isAlive)
					if isAlive.Alive {
						neighbours++
					}

				} else if world[y+i][x+j] == alive {
					neighbours++
				}
			}
		}
	}
	return neighbours
}
func calculateNextState() [][]byte {
	height := endY - startY
	width := endX - startX
	newWorld := makeWorld(height, width)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			neighbours := calculateNeighbours(startX+x, startY+y, height, width)
			//call CellFlipped event when a cell state is changed
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

type Client struct{}

func (c *Client) Calculate(req stubs.Request, res *stubs.CalculatedValues) (err error) {
	res.World = calculateNextState()
	res.AliveCells = calculateAliveCells()
	res.CellFlipped = cellFlipped
	fmt.Println("Calculated")

	return
}
func (c *Client) Neighbour(req stubs.NeighbourAddr, res *stubs.Response) (err error) {
	nextNeighbourAddr = req.NextAddr
	previousNeighbourAddr = req.PreviousAddr
	fmt.Println("Get Neighbours")

	return
}
func (c *Client) GetClientWorld(req stubs.ClientValues, res *stubs.Response) (err error) {
	world = req.World
	startY = req.StartX
	endY = req.EndY
	startX = req.EndX
	endX = req.EndX
	fmt.Println("Get World")
	return
}
func (c *Client) GetEdgeValue(req stubs.Coordinate, res *stubs.IsAlive) (err error) {
	if req.Y < 0 {
		if world[endY][req.X] == alive {
			res.Alive = true
		} else {
			res.Alive = false

		}
	} else if req.Y > 0 {
		if world[0][req.X] == alive {
			res.Alive = true
		} else {
			res.Alive = false
		}
	}
	fmt.Println("Get edge value")

	return
}
func main() {
	pAddr := flag.String("ip", "192.168.148.174:8030", "IP and port to listen on")
	distributorAddr := flag.String("distributor", "192.168.148.174:8050", "Address of distributor instance")
	flag.Parse()
	listener, err := net.Listen("tcp", *pAddr)
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
	}
	defer listener.Close()

	client, err := rpc.Dial("tcp", *distributorAddr)
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
	}
	defer client.Close()

	Connect := stubs.Client{ClientAddr: *pAddr}
	response := new(stubs.Response)
	client.Call(stubs.ConnectToDistributor, Connect, response)
	rpc.Register(&Client{})
	rpc.Accept(listener)

}
