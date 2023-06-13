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
const packetSize = 96
const dataBufferLength = packetSize
const audioBufferLength = dataBufferLength / wordLength
const sampleRate = 48000

var sendCount int32

func cleanup(conn net.Conn) {
	conn.Close()
	portaudio.Terminate()
}


func counter() {
	for {

		time.Sleep(1 * time.Second)
		fmt.Printf("TX: %6d\n", sendCount)
		sendCount = 0
	}
}

func main() {
	dataBuffer := make([]byte, dataBufferLength)
	audioBuffer := make([]float32, audioBufferLength)

	chk(portaudio.Initialize())
	defer portaudio.Terminate()

	stream, err := portaudio.OpenDefaultStream(channels, 0, sampleRate, len(audioBuffer), audioBuffer)
	chk(err)
	defer stream.Close()
	chk(stream.Start())



	conn, err := net.Dial("udp", "224.0.0.3:1234")
	chk(err)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-c
		fmt.Println("Exiting")
        cleanup(conn)
        os.Exit(0)
    }()

	go counter()

	sampleFactor := float32(0.1)
	saw := float32(-1.0)

	for {
		chk(stream.Read())
		sendCount++
		for i := range audioBuffer {
			/* sine := math.Sin(phase * 2.0 * math.Pi)
			_, phase = math.Modf(phase + sampleFactor) */
			/* if i > (audioBufferLength / 2) {
				audioBuffer[i] = saw
			} else {
				audioBuffer[i] = 0.0
			}*/

			short := int16(audioBuffer[i] * 32767)
			dataBuffer[i * 2] = byte(short & 0xFF);
			dataBuffer[(i * 2) +1] = byte(short >> 8);
		}
		_,err := conn.Write(dataBuffer)
		chk(err)
		saw += sampleFactor
		if saw >= 1 {
			saw = -1
		}
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
