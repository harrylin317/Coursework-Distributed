package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ExecuteAllTurns = "DistributorOperation.ExecuteAllTurns"
var KeyPressed = "DistributorOperation.KeyPressed"
var InitializeValues = "DistributorOperation.InitializeValues"
var GetWorld = "DistributorOperation.GetWorld"
var GetCurrentState = "DistributorOperation.GetCurrentState"
var CheckIfInitialized = "DistributorOperation.CheckIfInitialized"
var GetFilename = "DistributorOperation.GetFilename"

var Test = "DistributorOperation.Test"

type World struct {
	World [][]byte
}

type RequiredValue struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}
type Initialized struct {
	Initialized bool
}
type Filename struct {
	Filename string
}

// type Turn struct {
// 	Turn int
// }

// type AliveCells struct {
// 	AliveCells []util.Cell
// }
type Response struct {
	Message string
}
type Request struct {
	Message string
}
type Key struct {
	Key rune
}
type State struct {
	Turn        int
	AliveCells  []util.Cell
	CellFlipped []util.Cell
}
