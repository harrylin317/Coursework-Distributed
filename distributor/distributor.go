package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"

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
type clientPack struct {
	address string
	client  *rpc.Client
}

var (
	world                                     [][]byte
	turn, imageHeight, imageWidth, totalTurns int
	pauseChan                                 = make(chan bool)
	terminateChan                             = make(chan bool)
	terminate                                 = false
	aliveCells, cellFlipped                   []util.Cell
	filename                                  string
	connectedToController                     = false
	connectionChan                            = make(chan bool)
	initialized                               = false
	spaceAvaliable, workAvaliable             semaphore.Semaphore
	mutex                                     *sync.Mutex
	executing                                 bool
	connectedClients                          = 0
	clientList                                []clientPack
	clientMutex                               = &sync.Mutex{}
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

type DistributorOperation struct{}

//Runs calculation on all turns
func (d *DistributorOperation) ExecuteAllTurns(req stubs.Request, res *stubs.Response) (err error) {
	setClientNeighbours()
	sendWorldToClients()
	for turn = 0; turn < totalTurns; turn++ {
		//select statement used to block loop when key is pressed
		select {
		case <-pauseChan:
			select {
			case <-pauseChan:
				turn--
				break
			case connectedToController = <-connectionChan:
				turn--
				break
			case terminate = <-terminateChan:
				turn--
				break
			}
		case connectedToController = <-connectionChan:
			fmt.Println("set connetion to false")
			turn--
			break
		case terminate = <-terminateChan:
			turn--
			break
		default:
			executing = true
			//if statement is used to block calculation when controller is connected so it can be able to properly get values
			if connectedToController {
				spaceAvaliable.Wait()
			}
			initializeClientEdge()
			startClientCalculation()
			if connectedToController {
				workAvaliable.Post()
			}
			executing = false

		}
		if terminate {
			fmt.Println("Terminating")
			shutDown()
			break
		}

	}
	fmt.Println("Finished calculating all turns, setting initialized to false")
	initialized = false
	return
}

//get world
func (d *DistributorOperation) GetWorld(req stubs.Request, res *stubs.World) (err error) {
	res.World = world
	if terminate {
		terminateChan <- true
	}
	return
}

//get filename
func (d *DistributorOperation) GetFilename(req stubs.Request, res *stubs.Filename) (err error) {
	res.Filename = filename
	return
}

//used by the clients(workers), distributor creates a list of connectd clients to be used for rpc calls
func (d *DistributorOperation) ConnectToDistributor(req stubs.Client, res *stubs.Response) (err error) {
	clientMutex.Lock()
	connectedClients++
	clientDial, _ := rpc.Dial("tcp", req.ClientAddr)
	newClient := clientPack{address: req.ClientAddr, client: clientDial}
	clientList = append(clientList, newClient)
	fmt.Println("Connected clients: ")
	fmt.Println(clientList)
	clientMutex.Unlock()

	return
}

//gets the current game state, semaphore used to block ExecuteAllTurns() in order to get values
func (d *DistributorOperation) GetCurrentState(req stubs.Request, res *stubs.State) (err error) {
	workAvaliable.Wait()
	res.Turn = turn
	res.AliveCells = aliveCells
	res.CellFlipped = cellFlipped
	spaceAvaliable.Post()
	return
}

//checks if the world is initialized, this is used when a controller is trying to reconnect
func (d *DistributorOperation) CheckIfInitialized(req stubs.Request, res *stubs.Initialized) (err error) {
	fmt.Println("check if inintilized")

	res.Initialized = initialized
	//the executing value prevents this function from changing connectedToController while ExecuteAllTurns is calculating,
	//otherwise, if it changes connectedToController mid turn, mutex gets unlocked when its already unlocked, resulting in error
	if !connectedToController {
		fmt.Println("not connected to controller")
		for {
			if !executing {
				fmt.Println("connected to controller")
				connectedToController = true
				break
			}
		}
	}
	return
}

//handles keypressing by sending down channels
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
	case 'k':
		terminateChan <- true
		res.Message = "Exit"
	}
	return
}

//initialize values inside distributor
func (d *DistributorOperation) InitializeValues(req stubs.RequiredValue, res *stubs.State) (err error) {
	fmt.Println("initializing values")
	initialized = true
	world = req.World
	imageHeight = req.ImageHeight
	imageWidth = req.ImageWidth
	totalTurns = req.Turns
	aliveCells = calculateAliveCells(imageHeight, imageWidth, world)
	res.AliveCells = aliveCells
	spaceAvaliable = semaphore.Init(1, 1)
	workAvaliable = semaphore.Init(1, 0)
	//used for benchmarking, limit the amount of clients used
	if req.LimitConnection != 0 {
		connectedClients = req.LimitConnection
	}

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
	newAliveCells := []util.Cell{}
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageHeight; x++ {
			if world[y][x] == alive {
				newAliveCells = append(newAliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return newAliveCells
}

//send neighbouring client ip address to all clients
func setClientNeighbours() {
	if connectedClients == 1 {
		return
	}
	for i := 0; i < connectedClients; i++ {
		nextIndex := i + 1
		previousIndex := i - 1
		if nextIndex == connectedClients {
			nextIndex = 0
		} else if previousIndex == -1 {
			previousIndex = connectedClients - 1
		}
		nextClientAddr := clientList[nextIndex].address
		previousClientAddr := clientList[previousIndex].address
		request := stubs.NeighbourAddr{PreviousAddr: previousClientAddr, NextAddr: nextClientAddr}
		response := new(stubs.Response)
		err := clientList[i].client.Call(stubs.Neighbour, request, response)
		if err != nil {
			fmt.Println("Error in setting neighbour")
			fmt.Println(err)
		}
	}

	return
}

//seperate the world evenly and sent to clients
func sendWorldToClients() {
	dividedLength := imageHeight / connectedClients
	checkRemainder := imageHeight % connectedClients
	clientImageHeight := dividedLength
	clientImageWidth := imageWidth

	tmp := 0
	for i := 0; i < connectedClients; i++ {
		if checkRemainder != 0 && i == connectedClients-1 {
			fmt.Println("Remainder")
			clientImageHeight = imageHeight - (dividedLength * i)
		}
		clientWorld := makeWorld(clientImageHeight, clientImageWidth)
		for y := 0; y < clientImageHeight; y++ {
			for x := 0; x < clientImageWidth; x++ {
				clientWorld[y][x] = world[y+tmp][x]
			}
		}
		clientValues := stubs.ClientValues{ImageHeight: clientImageHeight, ImageWidth: clientImageWidth, World: clientWorld}
		response := new(stubs.Response)
		clientList[i].client.Call(stubs.GetClientWorld, clientValues, response)
		tmp = tmp + dividedLength
	}

	return

}

//worker function that calls client to calculate world
func workerCalculate(client *rpc.Client, doneChannel chan stubs.CalculatedValues) {
	clientCalculatedValues := new(stubs.CalculatedValues)
	request := new(stubs.Request)

	err := client.Call(stubs.Calculate, request, clientCalculatedValues)
	if err != nil {
		fmt.Println("Error in workercalculate")
		fmt.Println(err)
	}
	doneChannel <- *clientCalculatedValues

}

//creates channels for workers and collects the finished world by appending them together.
func startClientCalculation() {
	cellFlipped = []util.Cell{}
	aliveCells = []util.Cell{}
	doneChannels := make([]chan stubs.CalculatedValues, connectedClients)
	for i := 0; i < connectedClients; i++ {
		doneChannels[i] = make(chan stubs.CalculatedValues)
		go workerCalculate(clientList[i].client, doneChannels[i])
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
	return
}

//sends edge (top/bottom row) of the client's world to its relative neighbours
func initializeClientEdge() {
	for i := 0; i < connectedClients; i++ {
		request := new(stubs.Request)
		response := new(stubs.Response)
		err := clientList[i].client.Call(stubs.SendEdgeValue, request, response)
		if err != nil {
			fmt.Println("Error in initialize edge")
			fmt.Println(err)
		}
	}
	return
}

//waits for controller to call final GetWorld() before terminating
//notifies clients to terminate, waits 3 seconds and terminates distributor
func shutDown() {
	<-terminateChan
	for _, currentClient := range clientList {
		request := new(stubs.Request)
		response := new(stubs.Response)
		err := currentClient.client.Call(stubs.Shutdown, request, response)
		if err != nil {
			fmt.Println("Error in shutting down")
			fmt.Println(err)
		}

	}
	fmt.Println("Shutting Down...")
	time.Sleep(time.Second * 3)

	os.Exit(0)

}
