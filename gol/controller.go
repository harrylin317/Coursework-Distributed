package gol

import (
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type controllerChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan string
	Input      chan uint8
	Output     chan uint8
	keyPresses <-chan rune
}

const alive = 255
const dead = 0

func controller(p Params, keyPresses <-chan rune, c controllerChannels) {
	if p.Addr == "" {
		p.Addr = "127.0.0.1:8050"

	}
	client, err := rpc.Dial("tcp", p.Addr)
	if err != nil {
		fmt.Println("Error")
		fmt.Println(err)
	}
	defer client.Close()
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	turn := 0
	pause := false
	exit := false
	Request := new(stubs.Request)
	var aliveCells []util.Cell
	var filename string

	//checks if distributor is initialized
	checkInitialization := new(stubs.Initialized)
	client.Call(stubs.CheckIfInitialized, Request, checkInitialization)
	//if not, initialize
	if !checkInitialization.Initialized {
		filename = strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
		c.ioCommand <- ioInput
		c.ioFilename <- filename
		for y := 0; y < p.ImageHeight; y++ {
			for x := 0; x < p.ImageWidth; x++ {
				world[y][x] = <-c.Input
				if world[y][x] == alive {
					eventCellFlipped := CellFlipped{CompletedTurns: 0, Cell: util.Cell{X: x, Y: y}}
					c.events <- eventCellFlipped
				}
			}
		}
		SendValue := stubs.RequiredValue{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns, World: world, LimitConnection: p.LimitConnection}
		newState := new(stubs.State)
		client.Call(stubs.InitializeValues, SendValue, newState)
		aliveCells = newState.AliveCells

	} else {
		//if yes then get current turn and cellflipped so the SDL visualization can operate
		fmt.Println("Second Entry")
		GetFilename := new(stubs.Filename)
		client.Call(stubs.GetFilename, Request, GetFilename)
		filename = GetFilename.Filename
		GetCurrentState := new(stubs.State)
		client.Call(stubs.GetCurrentState, Request, GetCurrentState)
		turn = GetCurrentState.Turn
		aliveCells = GetCurrentState.AliveCells
		for _, cell := range aliveCells {
			eventCellFlipped := CellFlipped{CompletedTurns: turn, Cell: cell}
			c.events <- eventCellFlipped
		}

	}
	//ticker function
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if pause {
					break
				}
				aliveCellsCount := len(aliveCells)
				eventAliveCellsCount := AliveCellsCount{CompletedTurns: turn, CellsCount: aliveCellsCount}
				c.events <- eventAliveCellsCount
			}
		}
	}()
	//keypressed are sent to distributor
	go func() {
		for {
			key := <-keyPresses
			keyPressed := stubs.Key{Key: key}
			Response := new(stubs.Response)
			client.Call(stubs.KeyPressed, keyPressed, Response)
			switch Response.Message {
			case "Output":
				generateOutputFile(c, filename, turn, p, client)
			case "Exit":
				exit = true
			case "Pause":
				if pause {
					pause = false
					fmt.Println("Continuing execution on turn: ", turn)
					c.events <- StateChange{turn - 1, Executing}

				} else {
					pause = true
					fmt.Println("Currently paused on turn: ", turn+1)
					c.events <- StateChange{turn, Paused}
				}
			}
		}

	}()
	//if not initialized, tell distributor to start execution
	if !checkInitialization.Initialized {
		Response := new(stubs.Response)
		client.Go(stubs.ExecuteAllTurns, Request, Response, nil)
	}
	//loop monitors the calculation in distributor and gets value back every turn
	for {
		if turn == p.Turns || exit {
			break
		} else if !pause {
			Request := new(stubs.Request)
			CurrentState := new(stubs.State)
			client.Call(stubs.GetCurrentState, Request, CurrentState)
			turn = CurrentState.Turn
			aliveCells = CurrentState.AliveCells
			cellFlipped := CurrentState.CellFlipped
			for _, cell := range cellFlipped {
				eventCellFlipped := CellFlipped{CompletedTurns: turn, Cell: cell}
				c.events <- eventCellFlipped
			}
			eventTurnComplete := TurnComplete{CompletedTurns: turn}
			c.events <- eventTurnComplete
		}

	}
	eventFinalTurnComplete := FinalTurnComplete{CompletedTurns: turn, Alive: aliveCells}
	c.events <- eventFinalTurnComplete
	generateOutputFile(c, filename, turn, p, client)

	ticker.Stop()
	done <- true

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	close(c.events)

}

func generateOutputFile(c controllerChannels, filename string, turn int, p Params, client *rpc.Client) {
	c.ioCommand <- ioOutput
	filename = filename + "x" + strconv.Itoa(turn)
	c.ioFilename <- filename
	Request := new(stubs.Request)
	GetWorld := new(stubs.World)
	client.Call(stubs.GetWorld, Request, GetWorld)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {

			c.Output <- GetWorld.World[y][x]
		}
	}
	eventImageOutputComplete := ImageOutputComplete{CompletedTurns: turn, Filename: filename}
	c.events <- eventImageOutputComplete
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}
