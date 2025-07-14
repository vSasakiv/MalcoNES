package main

import (
	"log"
	"os"
	"runtime/pprof"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/mappers"
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
	nestest := cartridge.ReadFromFile("./testFiles/megaman.nes")

	mapper := mappers.NewMapper(&nestest)

	memory.LoadCartridge(mapper)
	ppu.LoadCartridge(mapper)

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
	handleInput()

	// Emulation step
	cycles := 0
	for cycles < 29780 {
		tick()
		cycles += int(cpu.GetCpu().LastInstructionCycles)
	}

	// time.Sleep(time.Second)
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
	ppu.ExecuteLoopy(cpu.GetCpu().LastInstructionCycles * 3)
	// fmt.Println(cpu.GetCpu().TraceStatus())
}
