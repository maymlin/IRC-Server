package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
)

type user struct {
	username    string
	nickname    string
	password    string
	cur_channel string
	connection  net.Conn
	message     string
}

type irc struct {
	channelsMap map[string]map[*user]net.Conn
	usersMap    map[string]*user
	nicknameMap map[string]*user
}

func handleConnection(conn net.Conn, instance irc) {
	defer conn.Close()

	s := bufio.NewScanner(conn)

	// A. User log in

	inst, user := login(instance, conn, s)

	userPtr := inst.usersMap[user]

	// B. Server waits for and executes instructions and messages

	func() {
		io.WriteString(conn, "["+userPtr.cur_channel+"] "+userPtr.nickname+": ")
		for s.Scan() {
			text := s.Text()
			chMap := inst.channelsMap
			uMap := inst.usersMap
			nickMap := inst.nicknameMap

			if len(text) > 0 {
				switch arg := strings.SplitN(text, " ", 3); arg[0] {
				case "/nick": // Change nickname
					if len(arg) == 2 {
						if verifyInput(arg[1]) == false || len(arg[1]) == 0 {
							io.WriteString(conn, "Invalid nickname. Try again.\n")
						} else {
							nickOps(arg[1], nickMap, userPtr, true)
						}
					} else {
						io.WriteString(conn, "Command usage: /nick newNickName\n")
					}
				case "/join": // Makes the user join a channel
					if len(arg) == 2 {
						if verifyInput(arg[1]) == false || len(arg[1]) == 0 {
							io.WriteString(conn, "Invalid channel. Try again.\n")
						} else {
							leaveChannel(userPtr.cur_channel, chMap, uMap, user, userPtr, conn, s)
							joinChannel(arg[1], chMap, uMap, user, userPtr, conn, s)
						}
					} else {
						io.WriteString(conn, "Command usage: /join channelName\n")
					}
				case "/part": // Makes the user leave a channel
					if len(arg) == 1 {
						leaveChannel(userPtr.cur_channel, chMap, uMap, user, userPtr, conn, s)
						joinChannel("waiting", chMap, uMap, user, userPtr, conn, s)
					} else {
						io.WriteString(conn, "Command usage: /part\n")
					}
				case "/names": // Lists all users connected to the server
					if len(arg) == 1 {
						io.WriteString(userPtr.connection, "Users connected to the server: ")
						for _, u := range uMap {
							if u.connection != nil {
								io.WriteString(conn, u.nickname+" ")
							}
						}
						io.WriteString(conn, "\n")
					} else {
						io.WriteString(conn, "Command usage: /names")
					}
				case "/list": // Lists all channels in the server
					if len(arg) == 1 {
						io.WriteString(conn, "List of channels: ")
						for ch, _ := range chMap {
							io.WriteString(conn, ch+" ")
						}
						io.WriteString(conn, "\n")
					}
				case "/privmsg": // Send a message to another user or a channel
					if len(arg) == 3 {
						if verifyInput(arg[1]) == false || len(arg[1]) == 0 {
							io.WriteString(conn, "Invalid user/channel. Try again.\n")
						} else {
							if _, ok := nickMap[arg[1]]; ok {
								dst := nickMap[arg[1]].username
								msg := "\nPrivate message from (" + userPtr.nickname + "): " + arg[2] + "\n"
								privateMsg(dst, uMap, msg, user)
							} else if authChannel(inst.channelsMap, arg[1]) {
								msg := "\nChannel message from (" + userPtr.nickname + ") in [" + userPtr.cur_channel + "] channel: " + arg[2] + "\n"
								channelMsg(arg[1], chMap, uMap, msg, userPtr.connection)
							} else {
								io.WriteString(conn, "Invalid user/channel. Try again.\n")
							}
						}
					} else {
						io.WriteString(conn, "Command usage: /privmsg nickName|channelName [msg]\n")
					}

				case "/exit":
					if len(arg) == 1 {
						defer conn.Close()
						delete(inst.channelsMap[userPtr.cur_channel], userPtr)
						userPtr.cur_channel = ""
						userPtr.connection = nil
						userPtr.message = ""
						return
					}
				default:
					if verifyInput(arg[0]) {
						msg := "[" + userPtr.cur_channel + "] (" + userPtr.nickname + "): " + text + "\n"
						channelMsg(uMap[user].cur_channel, chMap, uMap, msg, conn)
					}

				}

			}
			io.WriteString(conn, "["+userPtr.cur_channel+"] "+userPtr.nickname+": ")
		}
	}()
}

// A) Log-in sequence

func login(inst irc, conn net.Conn, s *bufio.Scanner) (irc, string) {
	chMap := inst.channelsMap
	uMap := inst.usersMap
	nickMap := inst.nicknameMap

	io.WriteString(conn, "Enter username: ")
	s.Scan()
	uName := s.Text()

	for !verifyInput(uName) {
		io.WriteString(conn, "Invalid characters used. Try again. \nEnter a username: ")
		s.Scan()
		uName = s.Text()
	}

	if authUser(uMap, uName) { // User exists, check password
		io.WriteString(conn, "Enter password: ")
		s.Scan()
		passwd := s.Text()

		for passwd != uMap[uName].password {
			io.WriteString(conn, "Password incorrect. Try again.\n")
			io.WriteString(conn, "Enter password: ")
			s.Scan()
			passwd = s.Text()
		}

	} else { // User doesn't exist, create new user
		userPtr := createUser(uName, uMap)

		io.WriteString(conn, "Enter a nickname: ")
		s.Scan()
		newNick := s.Text()

		vi := verifyInput(newNick)
		au := authUser(nickMap, newNick)

		// Check if nickname contains invalid characters & if nickname already exists

		for vi == false || au == true {
			if !vi {
				io.WriteString(conn, "Invalid characters used. Try again.\nEnter a nickname: ")
			}
			if au {
				io.WriteString(conn, "Nickname already in use. Try again.\nEnter a nickname: ")
			}

			s.Scan()
			newNick = s.Text()
			vi = verifyInput(newNick)
			au = authUser(nickMap, newNick)
		}

		// Create new nickname

		nickOps(newNick, nickMap, userPtr, false)

		// New user creates a password

		io.WriteString(conn, "Enter a password: ")
		s.Scan()
		newPasswd := s.Text()

		for !verifyInput(newPasswd) {
			io.WriteString(conn, "Invalid characters used. Try again. \nEnter a password: ")
			s.Scan()
			newPasswd = s.Text()
		}

		uMap[uName].password = newPasswd
	}
	uMap[uName].connection = conn

	// Prompt user to enter a channel

	io.WriteString(conn, "Enter a channel name: ")
	s.Scan()
	ch := s.Text()

	userPtr := uMap[uName]

	joinChannel(ch, chMap, uMap, uName, userPtr, conn, s)

	return inst, uName
}

func main() {
	// Listen on TCP port 6667 on all available unicast and
	// anycast IP addresses of the local system
	ln, err := net.Listen("tcp", ":6667")
	if err != nil {
		log.Fatal(err)
	}

	defer ln.Close()

	instance := irc{
		channelsMap: make(map[string]map[*user]net.Conn),
		usersMap:    make(map[string]*user),
		nicknameMap: make(map[string]*user),
	}

	for {
		//Wait for a connection
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// Handle the connection in a netw goroutine
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently

		go handleConnection(conn, instance)
		// Echo all incoming data
	}
}
