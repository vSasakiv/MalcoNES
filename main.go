package main

import (
	"unsafe"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"

	"github.com/veandco/go-sdl2/sdl"
)

const CPUCLOCK = 21441960

func main() {
	runEmulator()
}

func runEmulator() {
	nestest := cartridge.ReadFromFile("./testFiles/supermario.nes")
	memory.LoadFromCartridge(nestest)
	ppu.LoadFromCartridge(nestest)
	cpu.GetCpu().Reset()

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Rom viewer", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256*4, 240*4, sdl.WINDOW_SHOWN)

	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_RGB24), sdl.TEXTUREACCESS_TARGET, 256, 240)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()

	// err = texture.Update(nil, unsafe.Pointer(&frame.PixelData), 256*3)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// renderer.Clear()
	// renderer.Copy(texture, nil, nil)
	// renderer.Present()

	running := true
	for running {
		for range 20 {
			tick()
		}
		// pixelBuffer := ppu.GetPpu().CurrentFrame.PixelData
		pixelBuffer := ppu.GetPpu().CurrentPixelBuffer
		err = texture.Update(nil, unsafe.Pointer(&pixelBuffer), 256*3)
		if err != nil {
			panic(err)
		}

		renderer.Copy(texture, nil, nil)
		renderer.Present()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
				ppu.HexDumpVram("./vram.txt")
				println("Quit")
				running = false
			}
		}
		// time.Sleep(46 * time.Nanosecond * time.Duration(cpu.GetCpu().LastInstructionCycles))
		// time.Sleep(time.Millisecond)
	}
	// cpu.NesTestLineByLine()
	// for {
	// 	tick()
	// }
}

func tick() {
	// fmt.Println(cpu.GetCpu().TraceStatus() + " " + ppu.GetPpu().TracePpuStatus())
	cpu.ExecuteNext()
	ppu.Execute(cpu.GetCpu().LastInstructionCycles * 3)
	// fmt.Println(ppu.GetPpu().TracePpuStatus())
}

func renderChrRom() {
	nestest := cartridge.ReadFromFile("./testFiles/pacman.nes")
	memory.LoadFromCartridge(nestest)
	ppu.LoadFromCartridge(nestest)
	cpu.GetCpu().Reset()

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Rom viewer", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256*4, 240*4, sdl.WINDOW_SHOWN)

	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_RGB24), sdl.TEXTUREACCESS_TARGET, 256, 240)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()

	frame := ppu.NewFrame()
	frame.RenderRomBank(0)

	err = texture.Update(nil, unsafe.Pointer(&frame.PixelData), 256*3)
	if err != nil {
		panic(err)
	}

	renderer.Clear()
	renderer.Copy(texture, nil, nil)
	renderer.Present()

	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
				println("Quit")
				running = false
			}
		}
	}
}
