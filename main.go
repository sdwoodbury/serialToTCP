package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/albenik/go-serial"
)

func read(port serial.Port, shouldQuit chan bool) {

	buff := make([]byte, 100)
	readSerial := func() {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			fmt.Println("\nEOF")
		}
		fmt.Printf("%v", string(buff[:n]))
	}

	for {
		select {
		case <-shouldQuit:
			port.Close()
			return
		default:
			readSerial()
		}
	}
}

func mainwork(shouldQuit chan bool) {
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	for _, portName := range ports {
		fmt.Printf("Found port: %v\n", portName)

		mode := &serial.Mode{
			BaudRate: 115200,
		}
		port, err := serial.Open(portName, mode)
		if err != nil {
			log.Fatal(err)
		}

		read(port, shouldQuit)
	}
}

func main() {
	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)

	shouldQuit := make(chan bool)
	go mainwork(shouldQuit)

	<-signalChan

	log.Print("os.Interrupt - shutting down...\n")

	shouldQuit <- true

	os.Exit(0)
}
