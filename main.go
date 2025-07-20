package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"time"
	"vsasakiv/nesemulator/apu"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/mappers"
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
)

var running bool

const cpuClockFrequency float64 = 1789773.00              // 1.789773 Mhz
const ppuClockFrequency float64 = cpuClockFrequency * 3.0 // ppu frequency is triple of cpu, used as base cycle
const apuClockFrequency float64 = cpuClockFrequency / 2.0 // apu frequency is half of cpu
const audioSampleRate float64 = 44100.00                  // standard 44.1Khz sample rate
const cyclesPerSample = 121.7532

// sampleRate 44100 / 60fps
const samplesPerFrame = 735

const (
	screenWidth  = 256
	screenHeight = 240
	scale        = 3
)

type Game struct {
	pixels      []byte
	audioBuffer []byte
	screen      *ebiten.Image
	audioPipe   *io.PipeWriter
}

var JoyPad1 *controller.JoyPad
var Mapper mappers.Mapper

func main() {
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	op := &oto.NewContextOptions{}
	op.SampleRate = int(audioSampleRate)
	op.ChannelCount = 1
	op.Format = oto.FormatSignedInt16LE

	otoCtx, ready, err := oto.NewContext(op)
	if err != nil {
		fmt.Println("Error initializing oto context\n")
		return
	}
	<-ready

	pipeReader, pipeWriter := io.Pipe()
	player := otoCtx.NewPlayer(pipeReader)
	player.SetBufferSize(1200 * 2)
	player.Play()

	defer player.Close()

	ebiten.SetWindowSize(screenWidth*scale, screenHeight*scale)
	ebiten.SetWindowTitle("My Emulator (debug)")

	// setup and load cartridge
	nestest := cartridge.ReadFromFile("./testFiles/supermario.nes")
	Mapper = mappers.NewMapper(&nestest)

	memory.LoadCartridge(Mapper)
	ppu.LoadCartridge(Mapper)

	cpu.GetCpu().Reset()
	ppu.GetPpu().Reset()
	apu.GetApu().Reset()

	JoyPad1 = controller.NewJoypad()
	memory.ConnectJoyPad1(JoyPad1)

	game := &Game{
		pixels:      make([]byte, screenWidth*screenHeight*4),
		audioBuffer: make([]byte, samplesPerFrame*2),
		screen:      ebiten.NewImage(screenWidth, screenHeight),
		audioPipe:   pipeWriter,
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

}

var audioRate float64 = 0

func (g *Game) Update() error {
	start := time.Now()
	handleInput()

	// Emulation step
	sampleCount := 0

	// sync to audio because it is easier
	for {
		tick()
		if audioRate >= cyclesPerSample {
			audioRate -= cyclesPerSample
			sample := apu.GenSample()
			binary.LittleEndian.PutUint16(g.audioBuffer[sampleCount*2:], uint16(sample))
			sampleCount++
			if sampleCount == samplesPerFrame {
				break
			}
		} else {
			audioRate++
		}
	}

	g.audioPipe.Write(g.audioBuffer)
	rgb := ppu.GetPpu().GetPixelData()
	convertRGB24ToRGBA(g.pixels, rgb)
	duration := time.Since(start)
	if duration > time.Second/60 {
		log.Printf("Slow frame detected: %v", duration)
	}
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

// Clock all of the emulator components
func tick() {
	cpu.Clock()
	apu.Clock()
	ppu.Clock()
	Mapper.Clock(ppu.GetPpuStatus())
	// fmt.Println(cpu.GetCpu().TraceStatus())
}
