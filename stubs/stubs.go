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
var ConnectToDistributor = "DistributorOperation.ConnectToDistributor"

var Calculate = "Client.Calculate"
var Neighbour = "Client.Neighbour"
var GetClientWorld = "Client.GetClientWorld"
var GetEdgeValue = "Client.GetEdgeValue"
var SendEdgeValue = "Client.SendEdgeValue"
var Shutdown = "Client.Shutdown"

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
type Client struct {
	ClientAddr string
}
type ClientValues struct {
	ImageHeight, ImageWidth int
	World                   [][]byte
}

type IsAlive struct {
	Alive bool
}
type NeighbourAddr struct {
	PreviousAddr string
	NextAddr     string
}
type CalculatedValues struct {
	World       [][]byte
	AliveCells  []util.Cell
	CellFlipped []util.Cell
}
type Edge struct {
	Type string
	Edge []byte
}
