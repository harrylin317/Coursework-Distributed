package client

// import (
// 	"net/rpc"

// 	"uk.ac.bris.cs/gameoflife/stubs"
// )

// func RunClient(port string) {

// 	client, _ := rpc.Dial("tcp", port)
// 	defer client.Close()
// 	request := stubs.RequestValue{Message: "client connecting"}
// 	RequiredValue := new(stubs.RequiredValue)
// 	var newWorld [][]byte
// 	client.Call(stubs.GetValues, request, RequiredValue)
// 	for turns := 0; turns < RequiredValue.Maxturns; turns++ {
// 		client.Call(stubs.Calculate, RequiredValue, newWorld)

// 	}

// }
