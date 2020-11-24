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

	client, err := rpc.Dial("tcp", "192.168.148.174:8050")
	if err != nil {
		fmt.Println("Error connecting")
	}
	defer client.Close()
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool)
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	Response := new(stubs.Response)
	Request := new(stubs.Request)
	CurrentTurn := new(stubs.Turn)
	//turn := 0
	AliveCells := new(stubs.AliveCells)
	//Test := new(stubs.Turn)

	// func() {
	// 	wait := client.Go(stubs.Test, Request, Test, nil)
	// 	fmt.Println(Test.Turn)
	// 	time.Sleep(2 * time.Second)
	// 	fmt.Println(Test.Turn)

	// 	<-wait.Done
	// 	fmt.Println(Test.Turn)

	// }()

	c.ioCommand <- ioInput
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
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

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				client.Call(stubs.GetCurrentTurn, Request, CurrentTurn)
				if CurrentTurn.Turn == 0 {
					break
				} else {
					client.Call(stubs.GetAliveCells, Request, AliveCells)
					aliveCellsCount := len(AliveCells.AliveCells)
					eventAliveCellsCount := AliveCellsCount{CompletedTurns: CurrentTurn.Turn, CellsCount: aliveCellsCount}
					c.events <- eventAliveCellsCount
				}

			}
		}
	}()

	// go func() {
	// 	for {
	// 		key := <-keyPresses
	// 		if key == 's' {
	// 			generateOutputFile(c, filename, CurrentTurn, p, client)
	// 		} else {
	// 			keyPressed := stubs.Key{Key: key}
	// 			client.Call(stubs.KeyPressed, keyPressed, Response)
	// 			c.events <- StateChange{turn, Paused}
	// 		}

	// 	}

	// }()

	SendValue := stubs.RequiredValue{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns, World: world}
	client.Call(stubs.InitializeValues, SendValue, Response)
	finishCall := client.Go(stubs.ExecuteAllTurns, Request, CurrentTurn, nil)

	// for turn = 0; turn < p.Turns; turn++ {
	// 	client.Call(stubs.ExecuteTurn, Request, Response)
	// 	client.Call(stubs.GetAliveCells, Request, AliveCells)
	// 	eventTurnComplete := TurnComplete{CompletedTurns: turn}
	// 	c.events <- eventTurnComplete
	// }

	tmp := 0
	finishedExecution := false
	for {
		select {
		case <-finishCall.Done:
			finishedExecution = true
			break
		default:
			client.Call(stubs.GetCurrentTurn, Request, CurrentTurn)
			if tmp != CurrentTurn.Turn {
				//client.Call(stubs.GetAliveCells, Request, AliveCells)
				eventTurnComplete := TurnComplete{CompletedTurns: CurrentTurn.Turn}
				c.events <- eventTurnComplete
				tmp = CurrentTurn.Turn
			}

		}
		if finishedExecution {
			break
		}
	}
	client.Call(stubs.GetCurrentTurn, Request, CurrentTurn)
	client.Call(stubs.GetAliveCells, Request, AliveCells)
	fmt.Println("CELLALIVE", p.ImageHeight, p.ImageWidth, p.Turns, AliveCells.AliveCells)
	eventFinalTurnComplete := FinalTurnComplete{CompletedTurns: CurrentTurn.Turn, Alive: AliveCells.AliveCells}
	c.events <- eventFinalTurnComplete
	generateOutputFile(c, filename, CurrentTurn, p, client)

	ticker.Stop()
	done <- true

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{CurrentTurn.Turn, Quitting}
	close(c.events)

}

func callCellFlippedEvent(updatedWorld [][]byte, oldWorld [][]byte, height, width, turns int, c controllerChannels) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if updatedWorld[y][x] != oldWorld[y][x] {
				eventCellFlipped := CellFlipped{CompletedTurns: turns, Cell: util.Cell{X: x, Y: y}}
				c.events <- eventCellFlipped
			}
		}
	}
}

func generateOutputFile(c controllerChannels, filename string, CurrentTurn *stubs.Turn, p Params, client *rpc.Client) {
	c.ioCommand <- ioOutput
	filename = filename + "x" + strconv.Itoa(CurrentTurn.Turn)
	c.ioFilename <- filename
	Request := new(stubs.Request)
	GetWorld := new(stubs.World)
	client.Call(stubs.GetWorld, Request, GetWorld)

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.Output <- GetWorld.World[y][x]
		}
	}
	eventImageOutputComplete := ImageOutputComplete{CompletedTurns: CurrentTurn.Turn, Filename: filename}
	c.events <- eventImageOutputComplete
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}
