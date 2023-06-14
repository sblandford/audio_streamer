package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const channels = 2
const wordLength = 2
const dataBufferLength = 8192
const packetSize = 96
const audioBufferLength = packetSize / wordLength
const sampleRate = 48000
const missReportRateSec = 10

var b buffers



type buffers struct {
	dataBuffer  chan []byte
	audioBuffer chan []int16
	hitCount	float32
	missCount	float32
}

var quitPlayOut = make(chan bool)
func playOut() {

	chk(portaudio.Initialize())

	stream, err := portaudio.OpenDefaultStream(0, channels, sampleRate, audioBufferLength / channels, func(out []int16) {
		select {
		case buf := <- b.audioBuffer:
			copy(out, buf)
			b.hitCount++
		default:
			buf := make([]int16, audioBufferLength)
			for i := range buf {
				buf[i] = 0
			}
			copy(out, buf)
			b.missCount++
		}
	})
	chk(err)

	chk(stream.Start())

	<-quitPlayOut

	stream.Close()
	portaudio.Terminate()
	println("Audio terminated")
}

var quitProcessPackets = make(chan bool)
func processPackets () {
	for {
		select {
		case <- quitProcessPackets:
			println("Processing stopped")
			return
		case buf := <- b.dataBuffer:
			localAudioBuffer := make([]int16, audioBufferLength)
			for i := range localAudioBuffer {
				short := int16(buf[i*2]) | (int16(buf[(i*2)+1]) << 8)
				localAudioBuffer[i] = short
			}
			b.audioBuffer <- localAudioBuffer
		}
	}
}

var quitFetchPackets = make(chan bool)
func fetchPackets() {

	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.3:1234")
	chk(err)
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	chk(err)
	conn.SetReadBuffer(dataBufferLength)

	for {
		select {
		case <- quitFetchPackets:
			conn.Close()
			println("Connection closed")
			return
		default:
			buf := make([]byte, dataBufferLength)
			_, _, err := conn.ReadFromUDP(buf)
			chk(err)
			b.dataBuffer <- buf
		}
	}
}

var quitMissCounter = make(chan bool)
func MissCounter(statsText *widget.Label) {
	for {
		select {
		case <- quitMissCounter:
			return
		default:
			if (int64(time.Now().Unix()) % missReportRateSec) == 0 {
				missPercent := (b.missCount * 100) / (b.hitCount + b.missCount)
				stats := fmt.Sprintf("Error rate: %06.3f%%", missPercent)
				fmt.Println(stats)
				statsText.SetText(stats)
				b.hitCount = 0
				b.missCount = 0
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	b.dataBuffer = make(chan []byte, dataBufferLength)
	b.audioBuffer = make(chan []int16, 300)

	end := make(chan struct{})
	c := make(chan os.Signal, 1)

	myApp := app.New()
	myWindow := myApp.NewWindow("UDP Audio Receiver")

	 // Define a welcome text centered
	 text := canvas.NewText("Audio Receiver", color.White)
	 text.Alignment = fyne.TextAlignCenter

	 statsText := widget.NewLabel("")
	 statsText.SetText("---")


    // Display a vertical box containing text and error stats
    box := container.NewVBox(
        text,
		statsText,
    )

	// Trap Window Close
	myWindow.SetCloseIntercept(func() {
		c <- syscall.SIGTERM
	})

	// Display our content
	myWindow.SetContent(box)

	go fetchPackets()
	go processPackets()
	go playOut()
	go MissCounter(statsText)


	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("\nExiting\n")
		quitMissCounter <- true
		quitFetchPackets <- true
		quitProcessPackets <- true
		quitPlayOut <- true
		time.Sleep(1 * time.Second)
		close(end)
		os.Exit(0)
	}()

	// Show window and run app
	myWindow.ShowAndRun()

	<-end
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
