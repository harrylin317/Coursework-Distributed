package main

import (
	"flag"
	"net"
	"net/rpc"
	"sync"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

var (
	nextNeighbourAddr       = ""
	previousNeighbourAddr   = ""
	topEdge, bottomEdge     []byte
	world                   [][]byte
	imageHeight, imageWidth int
	aliveCells, cellFlipped []util.Cell
	mutex                   = &sync.Mutex{}
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
func sendEdge() {
	response := new(stubs.Response)

	sendEdgePrevious := new(stubs.Edge)
	edgePrevious := make([]byte, imageWidth)
	edgePrevious = world[0]
	sendEdgePrevious.Edge = edgePrevious
	sendEdgePrevious.Type = "Top of Original"
	clientCall(previousNeighbourAddr, sendEdgePrevious, response)

	sendEdgeNext := new(stubs.Edge)
	edgeNext := make([]byte, imageWidth)
	edgeNext = world[imageHeight-1]
	sendEdgeNext.Edge = edgeNext
	sendEdgeNext.Type = "Bottom of Original"
	clientCall(nextNeighbourAddr, sendEdgeNext, response)
	return
}

func clientCall(neighbourAddr string, edge *stubs.Edge, response *stubs.Response) {
	client, _ := rpc.Dial("tcp", neighbourAddr)
	client.Call(stubs.GetEdgeValue, edge, response)
	defer client.Close()
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

func (c *Client) Calculate(req stubs.Request, res *stubs.CalculatedValues) (err error) {
	world = calculateNextState()
	res.World = world
	res.AliveCells = calculateAliveCells(world)
	res.CellFlipped = cellFlipped

	return
}
func (c *Client) Neighbour(req stubs.NeighbourAddr, res *stubs.Response) (err error) {
	nextNeighbourAddr = req.NextAddr
	previousNeighbourAddr = req.PreviousAddr
	return
}
func (c *Client) GetClientWorld(req stubs.ClientValues, res *stubs.Response) (err error) {
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth

	return
}
func (c *Client) SendEdgeValue(req stubs.Request, res *stubs.Response) (err error) {
	if nextNeighbourAddr != "" && previousNeighbourAddr != "" {
		sendEdge()
	}
	return
}

func (c *Client) GetEdgeValue(req stubs.Edge, res *stubs.Response) (err error) {
	if req.Type == "Bottom of Original" {
		topEdge = req.Edge
	} else if req.Type == "Top of Original" {
		bottomEdge = req.Edge
	}
	return
}
func main() {
	pAddr := flag.String("ip", "192.168.148.174:8030", "IP and port to listen on")
	distributorAddr := flag.String("distributor", "192.168.148.174:8050", "Address of distributor instance")
	flag.Parse()
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	client, _ := rpc.Dial("tcp", *distributorAddr)
	defer client.Close()
	Connect := stubs.Client{ClientAddr: *pAddr}
	response := new(stubs.Response)
	client.Call(stubs.ConnectToDistributor, Connect, response)
	rpc.Register(&Client{})
	rpc.Accept(listener)

}
