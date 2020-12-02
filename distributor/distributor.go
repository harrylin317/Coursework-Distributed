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
			initializeClientEdge()

			startClientCalculation()
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
	fmt.Println("Connected Client: ", connectedClients)
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
		fmt.Println("Next:", nextClientAddr)
		fmt.Println("previous:", previousClientAddr)

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
		client.Close()

	}
	return

}

func sendWorldToClients() {
	dividedLength := imageHeight / connectedClients
	checkRemainder := imageHeight % connectedClients
	clientImageHeight := dividedLength
	clientImageWidth := imageWidth
	tmp := 0
	// fmt.Println("First world")

	// for _, x := range world {
	// 	fmt.Println(x)
	// }
	for i, currentClient := range clientList {

		if checkRemainder != 0 && i == connectedClients-1 {
			fmt.Println("Remainder")
			clientImageHeight = imageHeight
		}
		clientWorld := makeWorld(clientImageHeight, clientImageWidth)

		for y := 0; y < clientImageHeight; y++ {
			for x := 0; x < clientImageWidth; x++ {
				clientWorld[y][x] = world[y+tmp][x]
			}
		}
		// fmt.Println("clientWorld")
		// for _, x := range clientWorld {
		// 	fmt.Println(x)
		// }
		clientValues := stubs.ClientValues{ImageHeight: clientImageHeight, ImageWidth: clientImageWidth, World: clientWorld}
		client, _ := rpc.Dial("tcp", currentClient.ClientAddr)

		response := new(stubs.Response)
		client.Call(stubs.GetClientWorld, clientValues, response)

		tmp = tmp + dividedLength
		client.Close()

	}

}
func workerCalculate(clientAddr string, doneChannel chan stubs.CalculatedValues) {
	clientCalculatedValues := new(stubs.CalculatedValues)
	request := new(stubs.Request)
	client, _ := rpc.Dial("tcp", clientAddr)

	client.Call(stubs.Calculate, request, clientCalculatedValues)

	client.Close()
	doneChannel <- *clientCalculatedValues
}
func startClientCalculation() {
	cellFlipped = []util.Cell{}
	aliveCells = []util.Cell{}
	doneChannels := make([]chan stubs.CalculatedValues, connectedClients)

	for i := 0; i < connectedClients; i++ {
		doneChannels[i] = make(chan stubs.CalculatedValues)
	}

	for i := 0; i < connectedClients; i++ {
		go workerCalculate(clientList[i].ClientAddr, doneChannels[i])
	}

	newWorld := makeWorld(0, 0)
	tmp := 0
	for i := 0; i < connectedClients; i++ {
		calculatedValues := <-doneChannels[i]
		newWorld = append(newWorld, calculatedValues.World...)
		for _, cell := range calculatedValues.AliveCells {
			cell = util.Cell{X: cell.X, Y: cell.Y + tmp}
			aliveCells = append(aliveCells, cell)
		}
		for _, cell := range calculatedValues.CellFlipped {
			cell = util.Cell{X: cell.X, Y: cell.Y + tmp}
			cellFlipped = append(cellFlipped, cell)

		}
		tmp = tmp + len(calculatedValues.World)

	}
	world = newWorld
	// fmt.Println("after turn")
	// for _, x := range world {
	// 	fmt.Println(x)
	// }
	//fmt.Println(aliveCells)
	return
}
func initializeClientEdge() {
	request := new(stubs.Request)
	response := new(stubs.Response)

	for _, currentClient := range clientList {
		client, _ := rpc.Dial("tcp", currentClient.ClientAddr)
		client.Call(stubs.SendEdgeValue, request, response)
		client.Close()
	}
	return
}
