package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"time"

	"image"
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

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func getFrame() image.Image {
	height := int(min(size[0], size[1]))
	width := height
	offset_x := (int(size[0]) - width) / 2
	offset_y := (int(size[1]) - height) / 2

	src, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		return nil
	}

	cropSize := image.Rect(
		offset_x,
		offset_y,
		offset_x+width,
		offset_y+height,
	)
	croppedImage := src.(SubImager).SubImage(cropSize)

	return croppedImage
}

//go:embed mask.bmp
var mask_bytes []byte

func main() {
	initCamera()

	go updateFrames()

	defer camera.StopStreaming()
	defer camera.Close()

	// SDL3 window below
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

	mask_bmpStream, err := sdl.IOFromBytes(mask_bytes)
	if err != nil {
		panic(err)
	}

	mask_surface, err := sdl.LoadBMP_IO(mask_bmpStream, true)
	if err != nil {
		panic(err)
	}

	mask_texture, err := renderer.CreateTextureFromSurface(mask_surface)
	if err != nil {
		panic(err)
	}

	mask_texture.SetAlphaMod(64)

	renderer.SetDrawColor(255, 255, 255, 255)

	var draw_mask = false

	sdl.RunLoop(func() error {

		src := getFrame()

		var event sdl.Event

		for sdl.PollEvent(&event) {
			if event.Type == sdl.EVENT_QUIT {
				return sdl.EndLoop
			} else if event.Type == sdl.EVENT_KEY_DOWN {
				if event.KeyboardEvent().Key == sdl.K_RETURN {
					draw_mask = true
				}
			} else if event.Type == sdl.EVENT_KEY_UP {
				if event.KeyboardEvent().Key == sdl.K_RETURN {
					draw_mask = false

					dirname, err := os.UserHomeDir()
					if err != nil {
						panic(err)
					}

					filename := dirname + "/Pictures/takepic/" + time.Now().Format("2006-01-02") + ".jpg"

					buffer := bytes.NewBuffer([]byte{})

					err = jpeg.Encode(buffer, src, &jpeg.Options{
						Quality: 100,
					})
					if err != nil {
						panic(err)
					}

					os.MkdirAll(dirname+"/Pictures/takepic", 0755)
					os.WriteFile(filename, buffer.Bytes(), 0666)

					continue
				}
			}
		}

		buffer := bytes.NewBuffer([]byte{})

		err = bmp.Encode(buffer, src)
		if err != nil {
			panic(err)
		}

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

		dstRect.X = float32(0)
		dstRect.Y = float32(0)
		dstRect.W = float32(720)
		dstRect.H = float32(720)

		renderer.RenderTexture(texture, nil, &dstRect)

		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.DebugText(2, 2, time.Now().String())
		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.DebugText(0, 0, time.Now().String())

		if draw_mask {
			renderer.RenderTexture(mask_texture, nil, &dstRect)
		}

		renderer.Present()

		return nil
	})
}
