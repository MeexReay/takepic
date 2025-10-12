package main

import (
	"bytes"
	"fmt"
	// "io"
	"os"

	"image/jpeg"

	"golang.org/x/image/bmp"

	"github.com/Zyko0/go-sdl3/bin/binsdl"
	"github.com/Zyko0/go-sdl3/sdl"

	"github.com/blackjack/webcam"
)

var (
	camera      *webcam.Webcam
	frame       []byte
	size        [2]uint32
	format      webcam.PixelFormat
	format_name string
)

const YUYV = 1448695129 // YUYV 4:2:2
const MJPG = 1196444237 // Motion-JPEG

func initCamera() {
	var err error

	// Open webcam
	camera, err = webcam.Open("/dev/video0")
	if err != nil {
		panic(err.Error())
	}

	for format, format_name = range camera.GetSupportedFormats() {
		if format == MJPG {
			break
		}
	}

	fmt.Println("camera format:", format_name, format)

	cam_size := camera.GetSupportedFrameSizes(format)[0]
	size = [2]uint32{cam_size.MaxWidth, cam_size.MaxHeight}

	fmt.Println("camera frame:", size[0], size[1])

	camera.SetImageFormat(format, size[0], size[1])
	camera.SetBufferCount(30)
	camera.SetFramerate(30)

	fmt.Println("buffer count: 30")
	fmt.Println("framerate fps: 30")

	// Start streaming
	err = camera.StartStreaming()
	if err != nil {
		panic(err.Error())
	}
}

func updateFrames() {
	for {
		var err error
		err = camera.WaitForFrame(uint32(5))

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprint(os.Stderr, err.Error())
			continue
		default:
			panic(err.Error())
		}

		frame, err = camera.ReadFrame()
		if err != nil {
			panic(err.Error())
		}
	}
}

// func loadYuyv

func main() {
	initCamera()

	go updateFrames()

	defer camera.Close()

	defer binsdl.Load().Unload() // sdl.LoadLibrary(sdl.Path())
	defer sdl.Quit()

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}

	window, renderer, err := sdl.CreateWindowAndRenderer("takepic", 720, 720, 0)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()
	defer window.Destroy()

	renderer.SetDrawColor(255, 255, 255, 255)

	sdl.RunLoop(func() error {
		var event sdl.Event

		for sdl.PollEvent(&event) {
			if event.Type == sdl.EVENT_QUIT {
				return sdl.EndLoop
			}
		}

		image, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			return err
		}

		buffer := bytes.NewBuffer([]byte{})

		err = bmp.Encode(buffer, image)

		bmpStream, err := sdl.IOFromBytes(buffer.Bytes())
		if err != nil {
			panic(err)
		}

		surface, err := sdl.LoadBMP_IO(bmpStream, true)
		if err != nil {
			panic(err)
		}

		texture, err := renderer.CreateTextureFromSurface(surface)
		if err != nil {
			panic(err)
		}

		var dstRect sdl.FRect

		dstRect.X = float32(720-texture.W) / 2
		dstRect.Y = float32(720-texture.H) / 2
		dstRect.W = float32(texture.W)
		dstRect.H = float32(texture.H)
		renderer.RenderTexture(texture, nil, &dstRect)

		renderer.DebugText(50, 50, "Hello world")
		renderer.Present()

		return nil
	})
}
