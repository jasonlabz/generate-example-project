package main

import (
	"context"
	"log"

	"github.com/jasonlabz/generate-example-project/bootstrap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bootstrap.MustInit(ctx)
	log.Println("tools/fix command started")
	log.Println("tools/fix command finished")
}
