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
	//mutex                   *sync.Mutex
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
	//fmt.Println("Sending Edge")
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
	//fmt.Println("finished sending edge")

	return
}

func calculateAliveCells(word [][]byte) []util.Cell {
	//fmt.Println("calculating alive cell")

	aliveCells = []util.Cell{}
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			if world[y][x] == alive {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	//fmt.Println("fininshed calculating alive cell")

	return aliveCells
}
func calculateNeighbours(x, y, imageHeight, imageWidth int) int {
	//fmt.Println("calculating neighbour")

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
	//fmt.Println("finished calculating neighbour")

	return neighbours
}
func calculateNextState() [][]byte {
	//fmt.Println("calculating next state")

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
	//fmt.Println("finished calculating next state")

	return newWorld
}

type Client struct{}

func (c *Client) Calculate(req stubs.Request, res *stubs.CalculatedValues) (err error) {
	//fmt.Println("Calculating")

	world = calculateNextState()
	res.World = world
	res.AliveCells = calculateAliveCells(world)
	res.CellFlipped = cellFlipped
	//fmt.Println("fininshed calculate")

	return
}
func (c *Client) Neighbour(req stubs.NeighbourAddr, res *stubs.Response) (err error) {
	fmt.Println("getting neighbour")

	nextNeighbourAddr = req.NextAddr
	previousNeighbourAddr = req.PreviousAddr
	nextClient, _ = rpc.Dial("tcp", nextNeighbourAddr)
	previousClient, _ = rpc.Dial("tcp", previousNeighbourAddr)
	fmt.Println("finished getting neighbour")

	return
}
func (c *Client) GetClientWorld(req stubs.ClientValues, res *stubs.Response) (err error) {
	fmt.Println("get client world called")
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	fmt.Println("get client world fininshed")

	return
}
func (c *Client) SendEdgeValue(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("Sneding edge")
	if nextNeighbourAddr != "" && previousNeighbourAddr != "" {
		sendEdge()
	}
	fmt.Println("fininshed sending")

	return
}

func (c *Client) GetEdgeValue(req stubs.Edge, res *stubs.Response) (err error) {
	fmt.Println("Get edge value")

	if req.Type == "Bottom of Original" {
		topEdge = req.Edge
	} else if req.Type == "Top of Original" {
		bottomEdge = req.Edge
	}
	fmt.Println("fininshed Get edge value")

	return
}
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
	//port := strings.Split(*pAddr, ":")
	listener, err := net.Listen("tcp", *pAddr)
	if err != nil {
		fmt.Println("err")
		fmt.Println(err)
	}
	defer listener.Close()
	client, _ := rpc.Dial("tcp", *distributorAddr)

	defer client.Close()
	Connect := stubs.Client{ClientAddr: *pAddr}
	response := new(stubs.Response)
	client.Call(stubs.ConnectToDistributor, Connect, response)
	rpc.Accept(listener)

}
