package main

import (
	"os"
	"runtime/pprof"
	"time"
	"unsafe"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"

	"github.com/veandco/go-sdl2/sdl"
)

const CPUCLOCK = 21441960

var running bool

func main() {
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	runEmulator()
}

func runEmulator() {

	nestest := cartridge.ReadFromFile("./testFiles/pacman.nes")

	memory.LoadFromCartridge(nestest)

	ppu.LoadFromCartridge(nestest)
	cpu.GetCpu().Reset()
	joyPad := controller.NewJoypad()
	memory.ConnectJoyPad1(joyPad)

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Rom viewer", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256*3, 240*3, sdl.WINDOW_SHOWN)

	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_RGB24), sdl.TEXTUREACCESS_STREAMING, 256, 240)
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

	// 60fps
	const frameInterval = time.Second / 60
	running = true
	for running {
		frameStart := time.Now()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			handleEvent(joyPad, event)
		}

		cycles := 0
		for cycles < 320000 {
			tick()
			cycles += int(cpu.GetCpu().LastInstructionCycles)
		}

		pixelBuffer := ppu.GetPpu().CurrentFrame.GetPixelDataAndUpdateStatus()

		pixels, _, err := texture.Lock(nil)
		if err != nil {
			panic(err)
		}
		copy(pixels, pixelBuffer)
		texture.Unlock()
		// err = texture.Update(nil, unsafe.Pointer(&pixelBuffer[0]), 256*3)

		renderer.Copy(texture, nil, nil)
		renderer.Present()

		// pixelBuffer := ppu.GetPpu().CurrentFrame.PixelData
		// pixelBuffer := ppu.GetPpu().CurrentPixelBuffer
		// err = texture.Update(nil, unsafe.Pointer(&pixelBuffer), 256*3)
		// if err != nil {
		// 	panic(err)
		elapsedTime := time.Since(frameStart)
		if elapsedTime < frameInterval {
			time.Sleep(frameInterval - elapsedTime)
		}
		// time.Sleep(46 * time.Nanosecond * time.Duration(cpu.GetCpu().LastInstructionCycles))
		// time.Sleep(time.Millisecond)
	}
	// cpu.NesTestLineByLine()
	// for {
	// 	tick()
	// }
}

func handleEvent(joyPad *controller.JoyPad, event sdl.Event) {
	switch t := event.(type) {
	case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
		ppu.HexDumpVram("./vram.txt")
		println("Quit")
		running = false

	case *sdl.KeyboardEvent:
		switch t.State {
		case sdl.PRESSED:
			handleKeyPress(joyPad, t.Keysym.Sym, true)
		case sdl.RELEASED:
			handleKeyPress(joyPad, t.Keysym.Sym, false)
		}
	}
}

func handleKeyPress(joyPad *controller.JoyPad, key sdl.Keycode, pressed bool) {
	var val uint
	if pressed {
		val = 1
	} else {
		val = 0
	}
	switch key {
	case sdl.K_a:
		joyPad.SetButtonStatus(controller.LEFT, val)
	case sdl.K_s:
		joyPad.SetButtonStatus(controller.DOWN, val)
	case sdl.K_d:
		joyPad.SetButtonStatus(controller.RIGHT, val)
	case sdl.K_w:
		joyPad.SetButtonStatus(controller.UP, val)
	case sdl.K_j:
		joyPad.SetButtonStatus(controller.A, val)
	case sdl.K_k:
		joyPad.SetButtonStatus(controller.B, val)
	case sdl.K_SPACE:
		joyPad.SetButtonStatus(controller.START, val)
	case sdl.K_z:
		joyPad.SetButtonStatus(controller.SELECT, val)
	}
}

func tick() {
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
