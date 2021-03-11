package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func netWriter(sensorToTcpChan chan []byte, sensorToTcpSocket net.Conn, shouldQuit chan bool) {
	for {
		select {
		case <-shouldQuit:
			return
		case toWrite := <-sensorToTcpChan:
			//sensorToTcpSocket.SetWriteDeadline(time.Now().Add(time.Second * 10))
			sensorToTcpSocket.Write(toWrite)
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}

func netReader(tcpToSensorChan chan []byte, tcpToSensorSocket net.Conn, shouldQuit chan bool) {
	buff := make([]byte, 128)
	for {
		select {
		case <-shouldQuit:
			return
		default:
			//tcpToSensorSocket.SetReadDeadline(time.Now().Add(time.Second * 10))
			numRead, err := tcpToSensorSocket.Read(buff)
			if err != nil {
				log.Fatal(err)
			}
			if numRead == 0 {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			tcpToSensorChan <- buff[:numRead]
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
	tcpToSensorPort, _ := 8002, 1  //strconv.Atoi(os.Args[3]) // the server has ports as follows: readSerial, writeSerial
	sensorToTcpPort := 8001

	tcpToSensorSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverAddress, tcpToSensorPort))
	if err != nil {
		log.Fatal(err)
	}

	sensorToTcpSocket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverAddress, sensorToTcpPort))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected to sockets")

	shouldQuit := make(chan bool)
	tcpToSensorChan := make(chan []byte)
	sensorToTcpChan := make(chan []byte)
	go readWrite(getPort(comPort), tcpToSensorChan, sensorToTcpChan, shouldQuit)
	go netWriter(sensorToTcpChan, sensorToTcpSocket, shouldQuit)
	go netReader(tcpToSensorChan, tcpToSensorSocket, shouldQuit)

	<-signalChan

	log.Print("os.Interrupt - shutting down...\n")

	shouldQuit <- true

	os.Exit(0)
}
