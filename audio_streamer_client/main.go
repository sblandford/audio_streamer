package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gordonklaus/portaudio"
)

const channels = 2
const wordLength = 2
const dataBufferLength = 8192
const packetSize = 96
const audioBufferLength = packetSize / wordLength
const sampleRate = 48000

var quit = make(chan bool)
var b buffers



type buffers struct {
	dataBuffer  []byte
	audioBuffer chan []int16
	lock        sync.Mutex
	rCount      int
	wCount      int
}

func playOut() {
	chk(portaudio.Initialize())
	defer portaudio.Terminate()


	stream, err := portaudio.OpenDefaultStream(0, channels, sampleRate, audioBufferLength / channels, func(out []int16) {
		b.lock.Lock()
		select {
		case buf := <- b.audioBuffer:
			copy(out, buf)
		default:
			buf := make([]int16, audioBufferLength)
			for i := range buf {
				buf[i] = 0
			}
			copy(out, buf)
		}
		b.lock.Unlock()
		b.wCount++
	})
	chk(err)
	chk(stream.Start())

	<-quit

	stream.Close()
	portaudio.Terminate()
	println("Audio terminated")
}

func fetchPackets() {

	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.3:1234")
	chk(err)
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	chk(err)
	conn.SetReadBuffer(dataBufferLength)

	phaseFactor := int16(4096)
	saw := int16(-32768)
	for {
		select {
		case <-quit:
			conn.Close()
			println("Connection closed")
			return
		default:
			localAudioBuffer := make([]int16, audioBufferLength)
			_, _, err := conn.ReadFromUDP(b.dataBuffer)
			// _, err := conn.Read(b.dataBuffer)
			chk(err)
			b.lock.Lock()
			saw += phaseFactor
			for i := range localAudioBuffer {
				short := int16(b.dataBuffer[i*2]) | (int16(b.dataBuffer[(i*2)+1]) << 8)
				// short := saw
				localAudioBuffer[i] = short
			}
			b.lock.Unlock()
			b.audioBuffer <- localAudioBuffer
			b.rCount++
		}
	}
}

func counter() {
	for {
		select {
		case <-quit:
			return
		default:
			time.Sleep(1 * time.Second)
			fmt.Printf("RX: %6d, OUT: %6d\n", b.rCount, b.wCount)
			b.rCount = 0
			b.wCount = 0
		}
	}
}

func main() {
	b.dataBuffer = make([]byte, dataBufferLength)
	// b.audioBuffer = make([]float32, audioBufferLength)
	b.audioBuffer = make(chan []int16, 300)

	var end = make(chan struct{})

	go fetchPackets()
	go playOut()
	go counter ()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Exiting")
		quit <- true

		time.Sleep(1 * time.Second)
		close(end)
		os.Exit(0)
	}()
	<-end
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
