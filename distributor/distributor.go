package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/ChrisGora/semaphore"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const alive = 255
const dead = 0

type item struct {
	turn                    int
	aliveCells, cellFlipped []util.Cell
}
type buffer struct {
	b []item
}

var (
	world                                     [][]byte
	turn, imageHeight, imageWidth, totalTurns int
	pauseChan                                 = make(chan bool)
	aliveCells, cellFlipped                   []util.Cell
	filename                                  string
	connectedToController                     = false
	connectionChan                            = make(chan bool)
	initialized                               = false
	spaceAvaliable                            semaphore.Semaphore
	workAvaliable                             semaphore.Semaphore
	itemBuffer                                buffer
	mutex                                     *sync.Mutex
	executing                                 bool
	connectedClients                          = 0
	clientList                                []stubs.Client
)

// distributor divides the work between workers and interacts with other goroutines.
func main() {
	pAddr := flag.String("ip", ":8050", "port to listen on")
	flag.Parse()
	listener, err := net.Listen("tcp", *pAddr)
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
	}
	defer listener.Close()
	rpc.Register(&DistributorOperation{})
	rpc.Accept(listener)

}
func newBuffer(size int) buffer {
	return buffer{
		b: make([]item, size),
	}
}
func (buffer buffer) get() item {
	x := buffer.b[0]
	return x
}
func (buffer buffer) put(x item) {
	buffer.b[0] = x
}

type DistributorOperation struct{}

func (d *DistributorOperation) ExecuteAllTurns(req stubs.Request, res *stubs.Response) (err error) {
	setClientNeighbours()
	sendWorldToClients()

	for turn = 0; turn < totalTurns; turn++ {
		select {
		case <-pauseChan:
			select {
			case <-pauseChan:
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
			executing = true
			if connectedToController {
				spaceAvaliable.Wait()
				mutex.Lock()

			}
			startClientCalculation()
			fmt.Println("Finsihed Calculating")
			if connectedToController {
				newItem := item{turn: turn + 1, aliveCells: aliveCells, cellFlipped: cellFlipped}
				itemBuffer.put(newItem)
				mutex.Unlock()
				workAvaliable.Post()
			}
			executing = false
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
func (d *DistributorOperation) ConnectToDistributor(req stubs.Client, res *stubs.Response) (err error) {
	fmt.Println("Connected")
	connectedClients++
	clientList = append(clientList, req)
	fmt.Println(clientList)

	return
}

func (d *DistributorOperation) GetCurrentState(req stubs.Request, res *stubs.State) (err error) {
	workAvaliable.Wait()
	mutex.Lock()
	getItem := itemBuffer.get()
	res.Turn = getItem.turn
	res.AliveCells = getItem.aliveCells
	res.CellFlipped = getItem.cellFlipped
	mutex.Unlock()
	spaceAvaliable.Post()

	return

}
func (d *DistributorOperation) CheckIfInitialized(req stubs.Request, res *stubs.Initialized) (err error) {
	fmt.Println("check inintilize")
	res.Initialized = initialized
	//the executing value prevents this function from changing connectedToController while ExecuteAllTurns is calculating,
	//otherwise, if it changes connectedToController mid turn, mutex gets unlocked when its already unlocked, resulting in error
	if !connectedToController {
		for {
			if !executing {
				connectedToController = true
				break
			}
		}
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
	fmt.Println("initialized")
	initialized = true
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	totalTurns = req.Turns
	aliveCells = calculateAliveCells(imageHeight, imageWidth, world)
	//res.Turn = turn
	res.AliveCells = aliveCells
	spaceAvaliable = semaphore.Init(1, 1)
	workAvaliable = semaphore.Init(1, 0)
	itemBuffer = newBuffer(1)
	mutex = &sync.Mutex{}

	return
}

//make empty new world with given dimension
func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
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

func setClientNeighbours() {
	if connectedClients == 1 {
		return
	}
	for i, currentClient := range clientList {
		nextIndex := i + 1
		previousIndex := i - 1
		if nextIndex == connectedClients {
			nextIndex = 0
		} else if previousIndex == -1 {
			previousIndex = connectedClients - 1
		}
		nextClientAddr := clientList[nextIndex].ClientAddr
		previousClientAddr := clientList[previousIndex].ClientAddr

		client, err1 := rpc.Dial("tcp", currentClient.ClientAddr)
		if err1 != nil {
			fmt.Println("Error")
			fmt.Println(err1)
		}
		request := stubs.NeighbourAddr{PreviousAddr: previousClientAddr, NextAddr: nextClientAddr}
		response := new(stubs.Response)
		err2 := client.Call(stubs.Neighbour, request, response)
		if err2 != nil {
			fmt.Println("Error")
			fmt.Println(err2)
		}

	}

}

func sendWorldToClients() {
	dividedLength := imageHeight / connectedClients
	checkRemainder := imageHeight % connectedClients

	for i, currentClient := range clientList {
		startY := i * dividedLength
		endY := (i + 1) * dividedLength
		startX := 0
		endX := imageWidth
		if checkRemainder != 0 && i == connectedClients-1 {
			endY = imageHeight
		}
		clientWorld := makeWorld(endY, endX)

		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				clientWorld[y][x] = world[y][x]
			}
		}
		clientValues := stubs.ClientValues{StartY: startY, EndY: endY, StartX: startX, EndX: endX, World: clientWorld}
		client, err1 := rpc.Dial("tcp", currentClient.ClientAddr)
		if err1 != nil {
			fmt.Println("Error")
			fmt.Println(err1)
		}
		response := new(stubs.Response)
		err2 := client.Call(stubs.GetClientWorld, clientValues, response)
		if err2 != nil {
			fmt.Println("Error")
			fmt.Println(err2)
		}

	}

}

func startClientCalculation() {
	cellFlipped = []util.Cell{}
	aliveCells = []util.Cell{}
	doneChannels := make([]chan *rpc.Call, connectedClients)
	clientCalculatedValues := make([]*stubs.CalculatedValues, connectedClients)
	for i := 0; i < connectedClients; i++ {
		doneChannels[i] = make(chan *rpc.Call, 1)
		calculatedValues := new(stubs.CalculatedValues)
		clientCalculatedValues[i] = calculatedValues
	}
	for i, currentClient := range clientList {
		request := new(stubs.Request)
		client, err1 := rpc.Dial("tcp", currentClient.ClientAddr)
		if err1 != nil {
			fmt.Println("Error")
			fmt.Println(err1)
		}
		client.Go(stubs.Calculate, request, clientCalculatedValues[i], doneChannels[i])

	}
	newWorld := makeWorld(0, 0)
	for i := 0; i < connectedClients; i++ {
		fmt.Println("Waiting")

		<-doneChannels[i]
		fmt.Println("client finished ")

		newWorld = append(newWorld, clientCalculatedValues[i].World...)
		aliveCells = append(aliveCells, clientCalculatedValues[i].AliveCells...)
		cellFlipped = append(aliveCells, clientCalculatedValues[i].CellFlipped...)

	}
	world = newWorld
	return
}
