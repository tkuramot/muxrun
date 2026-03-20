package main

import (
	"os"

	"github.com/tkuramot/muxrun/cmd"
)

func main() {
	app := cmd.NewApp()
	app.Run(os.Args)
}
