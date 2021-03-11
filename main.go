package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/albenik/go-serial"
)

func readWrite(port serial.Port, tcpToSensorChan chan []byte, sensorToTcpChan chan []byte, shouldQuit chan bool) {

	buff := make([]byte, 100)
	readSerial := func() {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
		}
		sensorToTcpChan <- buff[:n]
		fmt.Printf("%v", string(buff[:n]))
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
		}
	}
}

func netMgr(tcpToSensorChan chan []byte, sensorToTcpChan chan []byte, serverAddress string, sensorToTcpPort int, tcpToSensorPort int, shouldQuit chan bool) {

	sensorToTcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", serverAddress, sensorToTcpPort))
	if err != nil {
		log.Fatal(err)
	}

	tcpToSensorAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", serverAddress, tcpToSensorPort))
	if err != nil {
		log.Fatal(err)
	}

	sensorToTcpConn, err := net.DialTCP("tcp", nil, sensorToTcpAddr)
	defer sensorToTcpConn.Close()
	if err != nil {
		fmt.Printf("error opening sensorToTcp port")
		log.Fatal(err)
	}

	tcpToSensorConn, err := net.DialTCP("tcp", nil, tcpToSensorAddr)
	defer tcpToSensorConn.Close()
	if err != nil {
		fmt.Println("error opening tcpToSensor port")
		log.Fatal(err)
	}

	readTcpWriteSensor := func() {
		buf := make([]byte, 1024)
		reqLen, err := tcpToSensorConn.Read(buf)
		if err == nil {
			tcpToSensorChan <- buf[:reqLen]
		}
	}

	for {
		select {
		case <-shouldQuit:
			return
		case toWrite := <-sensorToTcpChan:
			sensorToTcpConn.Write(toWrite)
		default:
			readTcpWriteSensor()
		}
	}
}

func mainwork(shouldQuit chan bool, comPort string, serverAddress string, sensorToTcpPort int, tcpToSensorPort int) {
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	for _, portName := range ports {
		if portName != comPort {
			continue
		}
		fmt.Printf("Found port: %v\n", portName)

		mode := &serial.Mode{
			BaudRate: 115200,
		}
		port, err := serial.Open(portName, mode)
		if err != nil {
			log.Fatal(err)
		}

		tcpToSensorChan := make(chan []byte)
		sensorToTcpChan := make(chan []byte)
		go netMgr(tcpToSensorChan, sensorToTcpChan, serverAddress, tcpToSensorPort, sensorToTcpPort, shouldQuit)
		go readWrite(port, tcpToSensorChan, sensorToTcpChan, shouldQuit)
		break
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
	comPort := "COM16"             //os.Args[1]
	serverAddress := "192.168.7.2" //os.Args[2]
	tcpToSensorPort, _ := 8001, 1  //strconv.Atoi(os.Args[3]) // the server has ports as follows: readSerial, writeSerial
	sensorToTcpPort := tcpToSensorPort + 1
	shouldQuit := make(chan bool)
	go mainwork(shouldQuit, comPort, serverAddress, sensorToTcpPort, tcpToSensorPort)

	<-signalChan

	log.Print("os.Interrupt - shutting down...\n")

	shouldQuit <- true

	os.Exit(0)
}
