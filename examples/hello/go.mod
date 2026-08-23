module github.com/losinggeneration/rovel/examples/hello

go 1.26.6

require (
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/veandco/go-sdl2 v0.5.0-alpha.7.0.20250220045537-7f43f67a3a12 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.17.0 // indirect
)

require (
	github.com/losinggeneration/rovel v0.0.0-00010101000000-000000000000
	github.com/losinggeneration/rovel/backend/sdl v0.0.0-00010101000000-000000000000
)

replace github.com/losinggeneration/rovel => ../..

replace github.com/losinggeneration/rovel/backend/sdl => ../../backend/sdl
