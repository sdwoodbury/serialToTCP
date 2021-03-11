package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	client "github.com/WesleiRamos/tcp_client"
	serial "github.com/albenik/go-serial"
)

func readWrite(port serial.Port, tcpToSensorChan chan []byte, sensorToTcpChan chan []byte, shouldQuit chan bool) {

	buff := make([]byte, 128)
	readSerial := func() {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
		}
		sensorToTcpChan <- buff[:n]
		//fmt.Printf("%v", string(buff[:n]))
	}

	for {
		select {
		case <-shouldQuit:
			port.Close()
			return
		case incoming := <-tcpToSensorChan:
			port.Write(incoming)
		default:
			readSerial()
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func netMgr(tcpToSensorChan chan []byte, sensorToTcpChan chan []byte, tcpToSensorSocket *client.Connection, sensorToTcpSocket *client.Connection, shouldQuit chan bool) {

	sensorToTcpSocket.OnOpen(func() {
		fmt.Println("sensor to tcp socket connected")
	})

	tcpToSensorSocket.OnOpen(func() {
		fmt.Println("tcp to sensor socket connected")
	})

	tcpToSensorSocket.OnMessage(func(message []byte) {
		tcpToSensorChan <- message
	})

	sensorToTcpSocket.Listen()

	for {
		select {
		case <-shouldQuit:
			return
		case toWrite := <-sensorToTcpChan:
			if sensorToTcpSocket.Connected {
				sensorToTcpSocket.Write(toWrite)
			}
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func getPort(comPort string) serial.Port {
	mode := &serial.Mode{
		BaudRate: 115200,
	}
	port, err := serial.Open(comPort, mode)
	if err != nil {
		log.Fatal(err)
	}

	return port
}

func main() {
	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	comPort := "COM16"             //os.Args[1]
	serverAddress := "192.168.7.2" //os.Args[2]
	tcpToSensorPort, _ := 8001, 1  //strconv.Atoi(os.Args[3]) // the server has ports as follows: readSerial, writeSerial
	sensorToTcpPort := tcpToSensorPort + 1
	shouldQuit := make(chan bool)

	sensorToTcpSocket := client.New(fmt.Sprintf("%s:%d", serverAddress, sensorToTcpPort))
	tcpToSensorSocket := client.New(fmt.Sprintf("%s:%d", serverAddress, tcpToSensorPort))
	sensorPort := getPort(comPort)

	tcpToSensorChan := make(chan []byte)
	sensorToTcpChan := make(chan []byte)
	go readWrite(sensorPort, tcpToSensorChan, sensorToTcpChan, shouldQuit)
	go netMgr(tcpToSensorChan, sensorToTcpChan, sensorToTcpSocket, tcpToSensorSocket, shouldQuit)

	<-signalChan

	log.Print("os.Interrupt - shutting down...\n")

	shouldQuit <- true

	os.Exit(0)
}
