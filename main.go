package main

import (
	"log"
	"os"
	"runtime/pprof"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"

	"github.com/hajimehoshi/ebiten/v2"
)

var running bool

const (
	screenWidth  = 256
	screenHeight = 240
	scale        = 3
)

type Game struct {
	pixels []byte
	screen *ebiten.Image
}

var JoyPad1 *controller.JoyPad

func main() {
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	ebiten.SetWindowSize(screenWidth*scale, screenHeight*scale)
	ebiten.SetWindowTitle("My Emulator (debug)")

	// setup and load cartridge
	nestest := cartridge.ReadFromFile("./testFiles/pacman.nes")

	memory.LoadFromCartridge(nestest)

	ppu.LoadFromCartridge(nestest)
	cpu.GetCpu().Reset()
	JoyPad1 = controller.NewJoypad()
	memory.ConnectJoyPad1(JoyPad1)

	game := &Game{
		pixels: make([]byte, screenWidth*screenHeight*4),
		screen: ebiten.NewImage(screenWidth, screenHeight),
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

}

func (g *Game) Update() error {
	// Input handling (replace with your custom logic)
	handleInput()

	// Emulation step
	cycles := 0
	for cycles < 297800 {
		tick()
		cycles += int(cpu.GetCpu().LastInstructionCycles)
	}

	// Update pixels
	rgb := ppu.GetPpu().CurrentFrame.GetPixelData()
	convertRGB24ToRGBA(g.pixels, rgb)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.screen.WritePixels(g.pixels)
	screen.DrawImage(g.screen, nil)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func convertRGB24ToRGBA(dst []byte, src []byte) {
	for si, di := 0, 0; si < len(src); si, di = si+3, di+4 {
		dst[di] = src[si]
		dst[di+1] = src[si+1]
		dst[di+2] = src[si+2]
		dst[di+3] = 0xFF // Opaque alpha
	}
}

func handleInput() {
	// Directional inputs
	JoyPad1.SetButtonStatus(controller.LEFT, ifPressed(ebiten.KeyA))
	JoyPad1.SetButtonStatus(controller.DOWN, ifPressed(ebiten.KeyS))
	JoyPad1.SetButtonStatus(controller.RIGHT, ifPressed(ebiten.KeyD))
	JoyPad1.SetButtonStatus(controller.UP, ifPressed(ebiten.KeyW))

	// Buttons
	JoyPad1.SetButtonStatus(controller.A, ifPressed(ebiten.KeyJ))
	JoyPad1.SetButtonStatus(controller.B, ifPressed(ebiten.KeyK))
	JoyPad1.SetButtonStatus(controller.START, ifPressed(ebiten.KeySpace))
	JoyPad1.SetButtonStatus(controller.SELECT, ifPressed(ebiten.KeyZ))
}

func ifPressed(key ebiten.Key) uint {
	if ebiten.IsKeyPressed(key) {
		return 1
	}
	return 0
}

func tick() {
	cpu.ExecuteNext()
	ppu.Execute(cpu.GetCpu().LastInstructionCycles * 3)
	// fmt.Println(ppu.GetPpu().TracePpuStatus())
}

// func handleEvent(joyPad *controller.JoyPad, event sdl.Event) {
// 	switch t := event.(type) {
// 	case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
// 		ppu.HexDumpVram("./vram.txt")
// 		println("Quit")
// 		running = false
//
// 	case *sdl.KeyboardEvent:
// 		switch t.State {
// 		case sdl.PRESSED:
// 			handleKeyPress(joyPad, t.Keysym.Sym, true)
// 		case sdl.RELEASED:
// 			handleKeyPress(joyPad, t.Keysym.Sym, false)
// 		}
// 	}
// }
//
// func handleKeyPress(joyPad *controller.JoyPad, key sdl.Keycode, pressed bool) {
// 	var val uint
// 	if pressed {
// 		val = 1
// 	} else {
// 		val = 0
// 	}
// 	switch key {
// 	case sdl.K_a:
// 		joyPad.SetButtonStatus(controller.LEFT, val)
// 	case sdl.K_s:
// 		joyPad.SetButtonStatus(controller.DOWN, val)
// 	case sdl.K_d:
// 		joyPad.SetButtonStatus(controller.RIGHT, val)
// 	case sdl.K_w:
// 		joyPad.SetButtonStatus(controller.UP, val)
// 	case sdl.K_j:
// 		joyPad.SetButtonStatus(controller.A, val)
// 	case sdl.K_k:
// 		joyPad.SetButtonStatus(controller.B, val)
// 	case sdl.K_SPACE:
// 		joyPad.SetButtonStatus(controller.START, val)
// 	case sdl.K_z:
// 		joyPad.SetButtonStatus(controller.SELECT, val)
// 	}
// }

// func renderChrRom() {
// 	nestest := cartridge.ReadFromFile("./testFiles/pacman.nes")
// 	memory.LoadFromCartridge(nestest)
// 	ppu.LoadFromCartridge(nestest)
// 	cpu.GetCpu().Reset()
//
// 	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
// 		panic(err)
// 	}
// 	defer sdl.Quit()
//
// 	window, err := sdl.CreateWindow("Rom viewer", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 256*4, 240*4, sdl.WINDOW_SHOWN)
//
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer window.Destroy()
//
// 	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer renderer.Destroy()
//
// 	texture, err := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_RGB24), sdl.TEXTUREACCESS_TARGET, 256, 240)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer texture.Destroy()
//
// 	frame := ppu.NewFrame()
// 	frame.RenderRomBank(0)
//
// 	err = texture.Update(nil, unsafe.Pointer(&frame.PixelData), 256*3)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	renderer.Clear()
// 	renderer.Copy(texture, nil, nil)
// 	renderer.Present()
//
// 	running := true
// 	for running {
// 		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
// 			switch event.(type) {
// 			case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
// 				println("Quit")
// 				running = false
// 			}
// 		}
// 	}
// }
