package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gordonklaus/portaudio"
)

const channels = 2
const wordLength = 2
const packetSize = 96
const indexByte = packetSize
const dataBufferLength = packetSize
const audioBufferLength = packetSize / wordLength
const sampleRate = 48000

func cleanup(conn net.Conn) {
	conn.Close()
	portaudio.Terminate()
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

	for {
		chk(stream.Read())
		for i := range audioBuffer {
			short := int16(audioBuffer[i] * 32767)
			dataBuffer[i * 2] = byte(short & 0xFF);
			dataBuffer[(i * 2) +1] = byte(short >> 8);
		}

		_,err := conn.Write(dataBuffer)
		chk(err)
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
