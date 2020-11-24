package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var Calculate = "DistributorOperation.Calculate"
var GetValues = "DistributorOperation.getValues"
var KeyPressed = "DistributorOperation.KeyPressed"
var SendValues = "DistributorOperation.SendValues"
var GetAliveCells = "DistributorOperation.GetAliveCells"
var Test = "DistributorOperation.Test"

type World struct {
	World [][]byte
}

type RequiredValue struct {
	ImageHeight, ImageWidth, Turns int
	World                          [][]byte
}

type RequestValue struct {
	Message string
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
