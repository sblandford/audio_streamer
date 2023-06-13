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
const indexByte = packetSize
const dataBufferLength = packetSize + 1
const audioBufferLength = packetSize / wordLength
const sampleRate = 48000
const redundancy = 1
const packetPeriodNs = (1000000 * audioBufferLength) / (sampleRate * channels)

func cleanup(conn net.Conn) {
	conn.Close()
	portaudio.Terminate()
}


/* func counter() {
	for {

		time.Sleep(1 * time.Second)
		fmt.Printf("TX: %6d\n", sendCount)
		sendCount = 0
	}
} */

func main() {
	dataBuffer := make([]byte, dataBufferLength)
	audioBuffer := make([]float32, audioBufferLength)
	counter := byte(0)

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

	// go counter()

	for {
		chk(stream.Read())
		for i := range audioBuffer {
			short := int16(audioBuffer[i] * 32767)
			dataBuffer[i * 2] = byte(short & 0xFF);
			dataBuffer[(i * 2) +1] = byte(short >> 8);
		}
		dataBuffer[indexByte] = counter
		counter++
		// Resend same packet for redunancy
		for i := 0; i < redundancy; i++ {
			_,err := conn.Write(dataBuffer)
			chk(err)
			time.Sleep((packetPeriodNs / redundancy) * time.Nanosecond)
		}
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
