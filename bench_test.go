package main

import (
	"fmt"
	"testing"

	"uk.ac.bris.cs/gameoflife/gol"
)

func BenchmarkExecutionTime(b *testing.B) {
	//os.Stdout = nil
	testParams := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 100}
	for i := 0; i < b.N; i++ {
		for connectedClient := 1; connectedClient <= 10; connectedClient++ {
			testArgument := fmt.Sprint(connectedClient)
			b.Run(testArgument, func(b *testing.B) {
				testParams.LimitConnection = connectedClient
				events := make(chan gol.Event)
				gol.Run(testParams, events, nil)
				for range events {
				}

			})
		}
	}

}
