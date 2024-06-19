package main

import (
	_ "github.com/synternet/data-layer-sdk/pkg/dotenv"

	"gitlab.com/synternet/amberdm/publisher/wasmlisher/cmd"
)

func main() {
	cmd.Execute()
}
