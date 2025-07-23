package main

import (
	"encoding/binary"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"time"
	"unsafe"

	"vsasakiv/nesemulator/apu"
	"vsasakiv/nesemulator/cartridge"
	"vsasakiv/nesemulator/controller"
	"vsasakiv/nesemulator/cpu"
	"vsasakiv/nesemulator/mappers"
	"vsasakiv/nesemulator/memory"
	"vsasakiv/nesemulator/ppu"

	"github.com/veandco/go-sdl2/sdl"
)

var running bool

const cpuClockFrequency float64 = 1789773.00
const ppuClockFrequency float64 = cpuClockFrequency * 3.0
const apuClockFrequency float64 = cpuClockFrequency / 2.0
const audioSampleRate float64 = 44100.00
const cyclesPerSample = 121.7532
const samplesPerFrame = 735

const (
	screenWidth  = 256
	screenHeight = 240
	scale        = 3
)

type Game struct {
	pixels      []byte
	audioBuffer []byte
	audioChan   chan []byte
	renderer    *sdl.Renderer
	texture     *sdl.Texture
	window      *sdl.Window
	audioDevice sdl.AudioDeviceID
}

var JoyPad1 *controller.JoyPad
var Mapper mappers.Mapper

func main() {
	f, _ := os.Create("cpu.prof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// SDL Initialization
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO | sdl.INIT_EVENTS); err != nil {
		log.Fatal(err)
	}
	defer sdl.Quit()

	// Setup window and renderer
	window, err := sdl.CreateWindow("My Emulator (debug)", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		screenWidth*scale, screenHeight*scale, sdl.WINDOW_SHOWN)
	if err != nil {
		log.Fatal(err)
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log.Fatal(err)
	}
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, screenWidth, screenHeight)
	if err != nil {
		log.Fatal(err)
	}
	defer texture.Destroy()

	// Setup audio
	audioSpec := sdl.AudioSpec{
		Freq:     int32(audioSampleRate),
		Format:   sdl.AUDIO_F32LSB,
		Channels: 1,
		Samples:  512,
	}
	audioDevice, err := sdl.OpenAudioDevice("", false, &audioSpec, nil, 0)
	if err != nil {
		log.Fatalf("Failed to open audio device: %v", err)
	}
	defer sdl.CloseAudioDevice(audioDevice)
	sdl.PauseAudioDevice(audioDevice, false)

	game := &Game{
		pixels:      make([]byte, screenWidth*screenHeight*4),
		audioBuffer: make([]byte, samplesPerFrame*4),
		audioChan:   make(chan []byte, 10),
		renderer:    renderer,
		texture:     texture,
		window:      window,
		audioDevice: audioDevice,
	}

	go func() {
		const maxQueuedFrames = 2 // allow 2 frames of audio in queue (~30ms)

		frameSize := samplesPerFrame * 4 // 735 samples * 4 bytes
		for buf := range game.audioChan {
			if len(buf) == 0 {
				continue
			}
			if sdl.GetQueuedAudioSize(game.audioDevice) < uint32(maxQueuedFrames*frameSize) {
				err := sdl.QueueAudio(game.audioDevice, buf)
				if err != nil {
					log.Println("SDL audio queue error:", err)
				}
			} else {
				// Drop frame or block — here we drop to avoid latency build-up
				// You can also: time.Sleep(time.Millisecond * 2)
			}
		}
	}()

	// Load cartridge and connect subsystems
	nestest := cartridge.ReadFromFile("./testFiles/supermario2usa.nes")
	Mapper = mappers.NewMapper(&nestest)

	memory.LoadCartridge(Mapper)
	ppu.LoadCartridge(Mapper)
	apu.GetApu().SetMapper(Mapper)

	cpu.GetCpu().Reset()
	ppu.GetPpu().Reset()
	apu.GetApu().Reset()

	JoyPad1 = controller.NewJoypad()
	memory.ConnectJoyPad1(JoyPad1)

	mainLoop(game)
}

func mainLoop(g *Game) {
	for {
		start := time.Now()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				return
			case *sdl.KeyboardEvent:
				handleInput(e)
			}
		}

		if err := g.Update(); err != nil {
			log.Println("Update error:", err)
			break
		}
		g.Draw()

		elapsed := time.Since(start)
		if elapsed < time.Second/60 {
			sdl.Delay(uint32((time.Second/60 - elapsed).Milliseconds()))
		}
	}
}

var audioRate float64 = 0

func (g *Game) Update() error {
	sampleCount := 0

	for {
		tick()
		if audioRate >= cyclesPerSample {
			audioRate -= cyclesPerSample
			sample := apu.GenSample()
			bs := math.Float32bits(sample)
			binary.LittleEndian.PutUint32(g.audioBuffer[sampleCount*4:], bs)
			sampleCount++
			if sampleCount == samplesPerFrame {
				break
			}
		} else {
			audioRate++
		}
	}

	buf := make([]byte, len(g.audioBuffer))
	copy(buf, g.audioBuffer)
	select {
	case g.audioChan <- buf:
	default:
	}

	rgb := ppu.GetPpu().GetPixelData()
	convertRGB24ToRGBA(g.pixels, rgb)
	return nil
}

func (g *Game) Draw() {
	if len(g.pixels) > 0 {
		g.texture.Update(nil, unsafe.Pointer(&g.pixels[0]), screenWidth*4)
	}
	g.renderer.Clear()
	g.renderer.Copy(g.texture, nil, nil)
	g.renderer.Present()
}

func convertRGB24ToRGBA(dst []byte, src []byte) {
	for si, di := 0, 0; si < len(src); si, di = si+3, di+4 {
		dst[di] = src[si]
		dst[di+1] = src[si+1]
		dst[di+2] = src[si+2]
		dst[di+3] = 0xFF
	}
}

func handleInput(event *sdl.KeyboardEvent) {
	pressed := event.State == sdl.PRESSED

	switch event.Keysym.Sym {
	case sdl.K_a:
		JoyPad1.SetButtonStatus(controller.LEFT, boolToUint(pressed))
	case sdl.K_s:
		JoyPad1.SetButtonStatus(controller.DOWN, boolToUint(pressed))
	case sdl.K_d:
		JoyPad1.SetButtonStatus(controller.RIGHT, boolToUint(pressed))
	case sdl.K_w:
		JoyPad1.SetButtonStatus(controller.UP, boolToUint(pressed))
	case sdl.K_j:
		JoyPad1.SetButtonStatus(controller.A, boolToUint(pressed))
	case sdl.K_k:
		JoyPad1.SetButtonStatus(controller.B, boolToUint(pressed))
	case sdl.K_SPACE:
		JoyPad1.SetButtonStatus(controller.START, boolToUint(pressed))
	case sdl.K_z:
		JoyPad1.SetButtonStatus(controller.SELECT, boolToUint(pressed))
	}
}

func boolToUint(b bool) uint {
	if b {
		return 1
	}
	return 0
}

func tick() {
	cpu.Clock()
	apu.Clock()
	ppu.Clock()
	Mapper.Clock(ppu.GetPpuStatus())
}
