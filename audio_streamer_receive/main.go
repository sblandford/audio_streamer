package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
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
const missReportRateSec = 10

var quit = make(chan bool)
var b buffers



type buffers struct {
	dataBuffer  chan []byte
	audioBuffer chan []int16
	hitCount	float32
	missCount	float32
}

func playOut() {
	chk(portaudio.Initialize())
	defer portaudio.Terminate()


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

	<-quit

	stream.Close()
	portaudio.Terminate()
	println("Audio terminated")
}

func processPackets () {
	for {
		select {
		case <-quit:
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

func fetchPackets() {

	addr, err := net.ResolveUDPAddr("udp4", "224.0.0.3:1234")
	chk(err)
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	chk(err)
	conn.SetReadBuffer(dataBufferLength)

	for {
		select {
		case <-quit:
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

func counter() {
	for {
		select {
		case <-quit:
			return
		default:
			time.Sleep(missReportRateSec * time.Second)
			missPercent := (b.missCount * 100) / (b.hitCount + b.missCount)
			fmt.Printf("Error rate: %06.3f%%\n", missPercent)
			b.hitCount = 0
			b.missCount = 0
		}
	}
}

func main() {
	b.dataBuffer = make(chan []byte, dataBufferLength)
	b.audioBuffer = make(chan []int16, 300)

	var end = make(chan struct{})

	go fetchPackets()
	go processPackets()
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
