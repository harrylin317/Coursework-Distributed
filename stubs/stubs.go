package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ExecuteAllTurns = "DistributorOperation.ExecuteAllTurns"
var KeyPressed = "DistributorOperation.KeyPressed"
var InitializeValues = "DistributorOperation.InitializeValues"
var GetAliveCells = "DistributorOperation.GetAliveCells"
var GetWorld = "DistributorOperation.GetWorld"
var GetCurrentTurn = "DistributorOperation.GetCurrentTurn"

var Test = "DistributorOperation.Test"

type World struct {
	World [][]byte
}

type RequiredValue struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

type Command struct {
	Message string
}

type Turn struct {
	Turn int
}

type AliveCells struct {
	AliveCells []util.Cell
}
type Response struct {
	Message string
}
type Request struct {
	Message string
}
type Key struct {
	Key rune
}
