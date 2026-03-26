//go:build sdl

package main

import (
	"github.com/losinggeneration/tui/backend/sdl"
)

func init() {
	var err error
	b, err := sdl.New(sdl.DefaultOptions())
	if err != nil {
		panic(err)
	}

	appOpts.Backend = b
}
