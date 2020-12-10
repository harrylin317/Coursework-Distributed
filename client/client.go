package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

var (
	nextNeighbourAddr          = ""
	previousNeighbourAddr      = ""
	nextClient, previousClient *rpc.Client
	topEdge, bottomEdge        []byte
	world                      [][]byte
	imageHeight, imageWidth    int
	aliveCells, cellFlipped    []util.Cell
	exitChan                   = make(chan bool)
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

//calls to the neighbourings client and sent its top and bottom edges
func sendEdge() {
	response := new(stubs.Response)
	sendEdgePrevious := new(stubs.Edge)
	edgePrevious := make([]byte, imageWidth)
	edgePrevious = world[0]
	sendEdgePrevious.Edge = edgePrevious
	sendEdgePrevious.Type = "Top of Original"
	err1 := previousClient.Call(stubs.GetEdgeValue, sendEdgePrevious, response)
	if err1 != nil {
		fmt.Println("Error")
		fmt.Println(err1)
	}
	sendEdgeNext := new(stubs.Edge)
	edgeNext := make([]byte, imageWidth)
	edgeNext = world[imageHeight-1]
	sendEdgeNext.Edge = edgeNext
	sendEdgeNext.Type = "Bottom of Original"
	err2 := nextClient.Call(stubs.GetEdgeValue, sendEdgeNext, response)
	if err2 != nil {
		fmt.Println("Error")
		fmt.Println(err2)
	}
	return
}

func calculateAliveCells(word [][]byte) []util.Cell {
	aliveCells = []util.Cell{}
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}

	return aliveCells
}

//when calculuating, if coordinate is out of bound, then refer to the edges sent by the neighbour client
func calculateNeighbours(x, y, imageHeight, imageWidth int) int {
	neighbours := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				if nextNeighbourAddr == "" || previousNeighbourAddr == "" {
					if world[mod(y+i, imageHeight)][mod(x+j, imageWidth)] == alive {
						neighbours++
					}
				} else if y+i == imageHeight {
					if bottomEdge[mod(x+j, imageWidth)] == alive {
						neighbours++
					}
				} else if y+i < 0 {

					if topEdge[mod(x+j, imageWidth)] == alive {
						neighbours++
					}

				} else if world[y+i][mod(x+j, imageWidth)] == alive {
					neighbours++
				}
			}
		}
	}

	return neighbours
}

func calculateNextState() [][]byte {
	cellFlipped = []util.Cell{}
	newWorld := makeWorld(imageHeight, imageWidth)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			neighbours := calculateNeighbours(x, y, imageHeight, imageWidth)
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

//calculates new state and sends the result
func (c *Client) Calculate(req stubs.Request, res *stubs.CalculatedValues) (err error) {

	world = calculateNextState()
	res.World = world
	res.AliveCells = calculateAliveCells(world)
	res.CellFlipped = cellFlipped

	return
}

//gets the neighbouring client IP address
func (c *Client) Neighbour(req stubs.NeighbourAddr, res *stubs.Response) (err error) {
	nextNeighbourAddr = req.NextAddr
	previousNeighbourAddr = req.PreviousAddr
	nextClient, _ = rpc.Dial("tcp", nextNeighbourAddr)
	previousClient, _ = rpc.Dial("tcp", previousNeighbourAddr)

	return
}

//gets the world
func (c *Client) GetClientWorld(req stubs.ClientValues, res *stubs.Response) (err error) {
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth

	return
}

//sends edges to neighbour
func (c *Client) SendEdgeValue(req stubs.Request, res *stubs.Response) (err error) {
	if nextNeighbourAddr != "" && previousNeighbourAddr != "" {
		sendEdge()
	}

	return
}

//gets the edge value
func (c *Client) GetEdgeValue(req stubs.Edge, res *stubs.Response) (err error) {
	//fmt.Println("calling get edge")
	if req.Type == "Bottom of Original" {
		topEdge = req.Edge
	} else if req.Type == "Top of Original" {
		bottomEdge = req.Edge
	}

	return
}

//terminates program, used channel here because we want the method to return first before terminating
func (c *Client) Shutdown(req stubs.Request, res *stubs.Response) (err error) {
	exitChan <- true
	return
}

func main() {
	pAddr := flag.String("ip", "127.0.0.1:8030", "IP and port to listen on")
	distributorAddr := flag.String("distributor", "127.0.0.1:8050", "Address of distributor instance")
	flag.Parse()
	go func() {
		<-exitChan
		fmt.Println("Shutting Down...")
		os.Exit(0)
	}()
	rpc.Register(&Client{})
	listener, err1 := net.Listen("tcp", *pAddr)
	if err1 != nil {
		fmt.Println("err")
		fmt.Println(err1)
	}
	defer listener.Close()
	client, err2 := rpc.Dial("tcp", *distributorAddr)
	if err2 != nil {
		fmt.Println("err")
		fmt.Println(err2)
	}
	defer client.Close()
	Connect := stubs.Client{ClientAddr: *pAddr}
	response := new(stubs.Response)
	client.Call(stubs.ConnectToDistributor, Connect, response)
	rpc.Accept(listener)

}
