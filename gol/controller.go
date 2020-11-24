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
	turns := 0
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	pauseChan := make(chan bool)
	exitChan := make(chan bool)
	//pause := false

	//aliveCellsCount := 0
	var aliveCellsSlice []util.Cell

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
	// 			if turns == 0 || pause {
	// 				break
	// 			}
	// 			aliveCellsRequiredValue := stubs.RequiredValue{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, World: world}
	// 				ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, World: world}
	// 			client.Call(stubs.GetAliveCells, aliveCellsRequiredValue, aliveCellsSlice)
	// 			aliveCellsCount = len(aliveCellsSlice)
	// 			eventAliveCellsCount := AliveCellsCount{CompletedTurns: turns, CellsCount: aliveCellsCount}
	// 			c.events <- eventAliveCellsCount
	// 		}
	// 	}
	// }()
	// go func() {
	// 	for {
	// 		select {
	// 		case key := <-keyPresses:
	// 			switch key {
	// 			case 's':
	// 				generateOutputFile(c, filename, turns, p, world)
	// 			case 'q':
	// 				exitChan <- true
	// 			case 'p':
	// 				if pause {
	// 					pause = false
	// 					fmt.Println("Continuing")
	// 					pauseChan <- pause
	// 				} else {
	// 					pause = true
	// 					c.events <- StateChange{turns, Executing}
	// 					pauseChan <- pause
	// 					c.events <- StateChange{turns, Paused}
	// 				}
	// 			}
	// 		}
	// 	}
	// }()

	newWorld := stubs.World{World: world}
	Response := new(stubs.Response)
	client.Call(stubs.SendValues, newWorld, Response)

	exit := false
	for turns = 0; turns < p.Turns; turns++ {
		select {
		case <-pauseChan:
			select {
			case <-pauseChan:
				break
			case exit = <-exitChan:
				break
			}
		case exit = <-exitChan:
		default:
			RequiredValue := stubs.RequiredValue{ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns, World: world}
			err := client.Call(stubs.Calculate, RequiredValue, Response)
			if err != nil {
				fmt.Println("RPC client returned error:")
				fmt.Println(err)
				return
			}
			// callCellFlippedEvent(updatedWorld.World, world, p.ImageHeight, p.ImageWidth, turns, c)
			// world = updatedWorld.World
			eventTurnComplete := TurnComplete{CompletedTurns: turns}
			c.events <- eventTurnComplete
		}
		if exit {
			break
		}

	}

	eventFinalTurnComplete := FinalTurnComplete{CompletedTurns: turns, Alive: aliveCellsSlice}
	c.events <- eventFinalTurnComplete

	generateOutputFile(c, filename, turns, p, world)

	ticker.Stop()
	done <- true

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turns, Quitting}
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

func generateOutputFile(c controllerChannels, filename string, turns int, p Params, world [][]byte) {
	c.ioCommand <- ioOutput
	filename = filename + "x" + strconv.Itoa(turns)
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.Output <- world[y][x]
		}
	}
	eventImageOutputComplete := ImageOutputComplete{CompletedTurns: turns, Filename: filename}
	c.events <- eventImageOutputComplete
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}
