package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	SERVER_HOST = "0.0.0.0"
	SERVER_PORT = "55555"
	SERVER_TYPE = "tcp"

	SERVER_UDP_PORT = 44444
)

type client struct {
	conn     net.Conn
	username string
}

// var isListenToClient bool = true
var clients []*client
var clearFunc map[string]func() //create a map for storing clear funcs

func init() {
	clearFunc = make(map[string]func()) //Initialize it
	clearFunc["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clearFunc["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func clearConsole() {
	osClearFunc, defined := clearFunc[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if defined {                                    //if we defined a clear func for that platform:
		osClearFunc() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

func getMessage(conn net.Conn) (string, error) {
	buffer := make([]byte, 1024)
	msgLen, err := conn.Read(buffer)
	if err != nil {
		return "", err
	}

	msg := string(buffer[:msgLen])
	msg = strings.TrimSpace(msg)
	msg = strings.TrimSuffix(msg, "\n")
	msg = strings.TrimSuffix(msg, "\r")
	return msg, nil
}

func removeClient(clients []*client, c client) []*client {
	// Get client's index in clients slice
	var s int
	for i, client := range clients {
		if client.conn == c.conn {
			s = i
		}
	}

	// Return new client slice without specified client
	return append(clients[:s], clients[s+1:]...)
}

func getServerStatus() string {
	serverStatus := fmt.Sprintf("\n----- [CLIENTS ON SERVER: %v] -----\n", len(clients))
	for i, c := range clients {
		serverStatus += fmt.Sprintf("[%d] : @%v\n", i+1, c.username)
	}

	serverStatus += "\n"
	return serverStatus
}

func getUsername(conn net.Conn) error {
	for {
		nameTaken := false
		conn.Write([]byte("[SERVER] Enter Username: "))
		msg, err := getMessage(conn)
		if err != nil {
			fmt.Printf("[ERROR READING MSG] %s\n", err)
			return err
		}

		msg = strings.ToLower(msg)

		for _, client := range clients {
			if client.username == msg {
				nameTaken = true
			}
		}

		// If name not taken => add client (conn+name) to clients list
		if !nameTaken {
			c := &client{conn: conn, username: msg}
			clients = append(clients, c)
			return nil
		}
	}
}

func isUserInLobby(name string) (*client, bool) {
	for _, client := range clients {
		if client.username == name {
			return client, true
		}
	}

	return nil, false
}

func handleClient(currentClient *client) {
	// Listen for client's messages
	for {

		msg, err := getMessage(currentClient.conn)
		if err != nil {
			// msg = fmt.Sprintf("[@%v Has Left The Lobby]", currentClient.username)
			clients = removeClient(clients, *currentClient)

			clearConsole()
			fmt.Println(getServerStatus())

			return
		}

		if msg == "?" {
			currentClient.conn.Write([]byte(getServerStatus()))
		} else if strings.Contains(msg, "=>") {
			validStructure := "<msg/command> => @<username>"
			msgStructure := strings.Split(msg, "=>")

			usernameArg := strings.Replace(strings.ReplaceAll(msgStructure[1], " ", ""), "@", "", 1)
			specifiedUser, isInLobby := isUserInLobby(usernameArg)
			if !isInLobby {
				currentClient.conn.Write([]byte(fmt.Sprintf("[SERVER] No User Named \"%s\" In Lobby...\n", specifiedUser)))
				continue
			}

			// "Hello There!" => @John
			if strings.Count(msgStructure[0], "\"") == 2 {
				// Get msg + user and send specified user the msg
				msg = strings.Split(msgStructure[0], "\"")[1]

				specifiedUser.conn.Write([]byte(fmt.Sprintf("[MSG FROM @%s] : %s\n", currentClient.username, msg)))
				currentClient.conn.Write([]byte("[SERVER] Message Sent Successfuly!\n"))

			} else if strings.ReplaceAll(msgStructure[0], " ", "") == "+" {
				specifiedUser.conn.Write([]byte(fmt.Sprintf("[SERVER] Request For Private Connection From @%s, Accept/Reject (a/r)? ", currentClient.username)))
				currentClient.conn.Write([]byte(fmt.Sprintf("[SERVER] Request For Private Connection Sent To @%s...\n", specifiedUser.username)))
				for {
					response, err := getMessage(specifiedUser.conn)
					if err != nil {
						clients = removeClient(clients, *currentClient)

						clearConsole()
						fmt.Println(getServerStatus())

						return
					}
					response = strings.ToLower(response)
					switch response {
					case "a":
						currentClient.conn.Write([]byte(fmt.Sprintf("[SERVER] @%s Accepted Invitation, Redirecting To Private Connection!\n", specifiedUser.username)))

						udpAddr := net.UDPAddr{IP: net.ParseIP(SERVER_HOST), Port: SERVER_UDP_PORT}
						udpConn, _ := net.ListenUDP("udp", &udpAddr)

						currentClient.conn.Write([]byte("+"))
						specifiedUser.conn.Write([]byte("+"))

						fmt.Println("Listening For Clients Udp Connections")

						i := 0
						for i <= 2 {
							buff := make([]byte, 1024)
							msgLen, clientUdpConn, err := udpConn.ReadFromUDP(buff)
							if err != nil {
								fmt.Println(err)
							}
							msg = string(buff[:msgLen])
							fmt.Printf("Got udp conn! msg: %s\n", msg)
							if msg == "+" {
								if string(clientUdpConn.String()) == currentClient.conn.RemoteAddr().String() {
									specifiedUser.conn.Write([]byte(fmt.Sprintf("=> %s", clientUdpConn.String())))
								} else {
									currentClient.conn.Write([]byte(fmt.Sprintf("=> %s", clientUdpConn.String())))
								}
							}
						}

						clients = removeClient(clients, *currentClient)
						clients = removeClient(clients, *specifiedUser)

						clearConsole()
						fmt.Println(getServerStatus())

						return
					case "r":
						currentClient.conn.Write([]byte(fmt.Sprintf("[SERVER] @%s Rejected Invitation...\n", specifiedUser.username)))
						break
					default:
						specifiedUser.conn.Write([]byte("[SERVER] Invalid Response, Please Try Again (a/r): "))
					}
				}
			} else {
				currentClient.conn.Write([]byte(fmt.Sprintf("[SERVER] Invalid request structure. valid structure: %s\n", validStructure)))
				continue
			}

		} else {
			msg = fmt.Sprintf("[SERVER] : No Proper Response For \"%s\"\n", msg)
			currentClient.conn.Write([]byte(msg))
		}
	}
}

func resolveNewConn(conn net.Conn) {
	defer conn.Close()
	err := getUsername(conn)
	if err != nil {
		fmt.Println("[ERROR] Couldn't get client's username")
		conn.Close()
		return
	}

	for _, c := range clients {
		if c.conn == conn {
			// fmt.Printf("[@%s HAS ENTERED THE LOBBY]\n", c.username)
			clearConsole()
			fmt.Print(getServerStatus())
			conn.Write([]byte(fmt.Sprintf("[SERVER] : Welcome to the lobby %s!\n", c.username)))
			handleClient(c)
		}
	}
}

func main() {
	server, err := net.Listen(SERVER_TYPE, SERVER_HOST+":"+SERVER_PORT)
	if err != nil {
		fmt.Printf("[ERROR OPENING SERVER] %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("[%s SERVER LISTENING ON %s:%s]\n", SERVER_TYPE, SERVER_HOST, SERVER_PORT)

	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Printf("[ERROR ACCEPTING CONNECTION] %s\n", err)
			conn.Close()
			continue
		}

		// fmt.Printf("[NEW CONNECTION]\n")

		// Get client's information and add to clients list
		go resolveNewConn(conn)
	}
}
