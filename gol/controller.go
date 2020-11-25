package gol

import (
	"fmt"
	"net/rpc"
	"strconv"

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
	// ticker := time.NewTicker(2 * time.Second)
	// done := make(chan bool)
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	Response := new(stubs.Response)
	Request := new(stubs.Request)
	CurrentTurn := new(stubs.Turn)
	turn := 0
	AliveCells := new(stubs.AliveCells)
	// Test := new(stubs.Turn)
	// Test.Turn = 12

	// func() {
	// 	fmt.Println(Test.Turn)

	// 	wait := client.Go(stubs.Test, Request, Test, nil)
	// 	client.Call(stubs.Test, Request, Test)

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

	// go func() {
	// 	for {
	// 		select {
	// 		case <-done:
	// 			return
	// 		case <-ticker.C:
	// 			client.Call(stubs.GetCurrentTurn, Request, CurrentTurn)
	// 			if CurrentTurn.Turn == 0 {
	// 				break
	// 			} else {
	// 				client.Call(stubs.GetAliveCells, Request, AliveCells)
	// 				aliveCellsCount := len(AliveCells.AliveCells)
	// 				eventAliveCellsCount := AliveCellsCount{CompletedTurns: CurrentTurn.Turn, CellsCount: aliveCellsCount}
	// 				c.events <- eventAliveCellsCount
	// 			}

	// 		}
	// 	}
	// }()

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
	client.Go(stubs.ExecuteAllTurns, Request, Response, nil)

	for {
		if turn == p.Turns {
			break
		}
		client.Call(stubs.GetCurrentTurn, Request, CurrentTurn)
		turn = CurrentTurn.Turn
		//fmt.Println((turn))
		eventTurnComplete := TurnComplete{CompletedTurns: turn}
		c.events <- eventTurnComplete

	}

	client.Call(stubs.GetAliveCells, Request, AliveCells)
	eventFinalTurnComplete := FinalTurnComplete{CompletedTurns: turn, Alive: AliveCells.AliveCells}
	c.events <- eventFinalTurnComplete
	generateOutputFile(c, filename, turn, p, client)

	// ticker.Stop()
	// done <- true

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
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
