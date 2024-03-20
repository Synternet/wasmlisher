package main

import (
	_ "github.com/syntropynet/data-layer-sdk/pkg/dotenv"

	"gitlab.com/syntropynet/amberdm/publisher/wasmlisher/cmd"
)

func main() {
	cmd.Execute()
}
