package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/derfenix/webarchive/adapters/processors"
	"github.com/derfenix/webarchive/config"
	"github.com/derfenix/webarchive/entity"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewDevelopment()
	procs, err := processors.NewProcessors(config.Config{}, log)
	if err != nil {
		panic(err)
	}

	page := entity.NewPage("https://en.wikipedia.org/wiki/March_18_Massacre", "", nil, nil, nil)
	meta, err := procs.GetMeta(context.Background(), page)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Meta: %+v\n", meta)
}
