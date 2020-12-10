package gol

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns           int
	Threads         int
	ImageWidth      int
	ImageHeight     int
	Addr            string
	LimitConnection int
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {
	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	ioFilename := make(chan string)
	Output := make(chan uint8)
	Input := make(chan uint8)

	controllerChannels := controllerChannels{
		events,
		ioCommand,
		ioIdle,
		ioFilename,
		Input,
		Output,
		keyPresses,
	}

	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: ioFilename,
		output:   Output,
		input:    Input,
	}

	go controller(p, keyPresses, controllerChannels)
	go startIo(p, ioChannels)

}
