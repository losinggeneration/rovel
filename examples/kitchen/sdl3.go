//go:build sdl3

package main

import (
	"github.com/losinggeneration/rovel/backend/sdl3"
)

func init() {
	var err error
	b, err := sdl3.New(sdl3.DefaultOptions())
	if err != nil {
		panic(err)
	}

	appOpts.Backend = b
}
