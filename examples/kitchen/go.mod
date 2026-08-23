module github.com/losinggeneration/rovel/examples/kitchen

go 1.26.6

require (
	github.com/losinggeneration/rovel v0.0.0-00010101000000-000000000000
	github.com/losinggeneration/rovel/backend/sdl v0.0.0-00010101000000-000000000000
	github.com/losinggeneration/rovel/backend/sdl3 v0.0.0-00010101000000-000000000000
)

require (
	github.com/Zyko0/go-sdl3 v0.1.1 // indirect
	github.com/Zyko0/purego-gen v0.0.1 // indirect
	github.com/ebitengine/purego v0.10.2 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/veandco/go-sdl2 v0.5.0-alpha.7.0.20250220045537-7f43f67a3a12 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.17.0 // indirect
)

replace github.com/losinggeneration/rovel => ../..

replace github.com/losinggeneration/rovel/backend/sdl => ../../backend/sdl

replace github.com/losinggeneration/rovel/backend/sdl3 => ../../backend/sdl3
