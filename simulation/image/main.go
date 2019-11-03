package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dedis/example-grpc/overlay"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("missing port")
	}

	o := overlay.NewOverlay(fmt.Sprintf(":%s", os.Args[1]))

	o.Serve()
}
