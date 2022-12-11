package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	SERVER_HOST     = "0.0.0.0"
	SERVER_PORT     = "55555"
	SERVER_UDP_PORT = "44444"
	SERVER_TYPE     = "tcp"
)

var (
	UDP_HOST = ""
	UDP_PORT = "44444"

	UDP_CONN        net.Conn
	UDP_REMOTE_ADDR *net.UDPAddr

	isUdpConnectionOpen bool = false
)

func readInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[ERROR READING INPUT] %s\n", err)
		return "", err
	}

	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, "\r", "")

	return input, nil
}

func getServerMsg(conn net.Conn) string {
	buffer := make([]byte, 1024)
	msgLen, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("[ERROR READING SERVER'S MSG] %s\n", err)
		os.Exit(1)
	}

	msg := string(buffer[:msgLen])
	return msg
}

func listenToServer(conn net.Conn) {
	for {
		msg := getServerMsg(conn)

		if strings.HasPrefix(msg, "+") {
			// buff := make([]byte, 2048)
			UDP_CONN, _ = net.Dial("udp", SERVER_HOST+":"+SERVER_UDP_PORT)
			// udpServerAddr, err := net.ResolveUDPAddr("udp", SERVER_HOST+":"+SERVER_UDP_PORT)
			// if err != nil {
			// 	fmt.Println("ResolveUDPAddr failed:", err.Error())
			// }
			conn.Write([]byte("+"))
			// UDP_CONN.WriteTo([]byte("+"), udpServerAddr)
			fmt.Printf("Sent + to udp server: %s\n", UDP_CONN.RemoteAddr())

		} else if strings.HasPrefix(msg, "=>") {
			msg = strings.TrimSpace(strings.Split(msg, "=>")[1])
			addr := strings.Split(msg, ":")
			UDP_HOST = addr[0]
			UDP_PORT = addr[1]
			UDP_REMOTE_ADDR, _ = net.ResolveUDPAddr("udp", UDP_HOST+":"+UDP_PORT)

			isUdpConnectionOpen = true
			conn.Close()

			return

		} else if strings.HasSuffix(strings.TrimSpace(msg), "?") {
			fmt.Printf("\r%v", msg)
			conn.Write([]byte("$"))
		} else {
			fmt.Printf("\r%v", msg)
		}
	}
}

func sendMsg(conn net.Conn, msg string) error {
	byteMsg := []byte(msg)
	_, err := conn.Write(byteMsg)
	if err != nil {
		fmt.Printf("[ERROR SENDING MSG] %v\n", err)
		return err
	}

	return nil
}

func serverHandler(conn net.Conn) {
	go listenToServer(conn)

	for {
		msg, err := readInput()
		if err != nil {
			continue
		}

		if isUdpConnectionOpen {
			return
		}

		err = sendMsg(conn, msg)
		if err != nil {
			break
		}
	}
}

func keepUdpConnAlive(udpConn net.Conn, addr *net.UDPAddr) {
	for {
		time.Sleep(5 * time.Second)
		udpConn.Write([]byte("0"))
	}
}

func listenForUdpPackets(udpConn net.Conn) {
	defer udpConn.Close()

	for {
		buff := make([]byte, 1024)
		msgLen, err := udpConn.Read(buff)
		if err != nil {
			continue
		}

		msg := string(buff[:msgLen])
		if msg == "0" {
			continue
		}

		fmt.Printf("\r%v", msg)
	}
}

func openUdpConn() net.Conn {
	// udpAddr, _ := net.ResolveUDPAddr("udp", SERVER_HOST+":")
	udpConn, err := net.Dial("udp", SERVER_HOST+":")
	if err != nil {
		fmt.Printf("[ERROR UDP ADDRES] %s\n", err)
		os.Exit(1)
	}

	return udpConn
}

func manageUdpConn() {
	// Hole Punching
	udpEndPoingAddr, _ := net.ResolveUDPAddr("udp", UDP_HOST+":"+UDP_PORT)
	UDP_CONN.Write([]byte("0"))

	go listenForUdpPackets(UDP_CONN)
	go keepUdpConnAlive(UDP_CONN, udpEndPoingAddr)

	for {
		msg, _ := readInput()
		byteMsg := []byte(msg)

		_, err := UDP_CONN.Write(byteMsg)
		if err != nil {
			fmt.Printf("[ERROR SENDING MSG] %v\n", err)
			continue
		}
	}
}

func main() {
	conn, err := net.Dial(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		fmt.Printf("[ERROR CONNECTING TO SERVER] %s\n", err)
		os.Exit(1)
	}

	defer conn.Close()
	fmt.Println("----- [CONNECTED TO SERVER SUCCESSFULY] -----")
	serverHandler(conn)

	if isUdpConnectionOpen {
		manageUdpConn()
	}

	// server, err := net.ResolveUDPAddr(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	// if err != nil {
	// 	fmt.Printf("[ERROR RESOLVING UDP ADDRES] %s\n", err)
	// 	os.Exit(1)
	// }

	// fmt.Printf("[%s SERVER LISTENING ON %s:%s]\n", SERVER_TYPE, SERVER_HOST, SERVER_PORT)

	// for {
	// 	conn, err := net.ListenUDP(SERVER_TYPE, server)
	// 	if err != nil {
	// 		fmt.Printf("[ERROR WITH INCOMING CONNECTION] %s", err)
	// 		conn.Close()
	// 		continue
	// 	}

	// }
}
