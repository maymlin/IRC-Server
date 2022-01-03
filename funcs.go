package main

import (
	"bufio"
	"io"
	"net"
)

// Input verification; only allow printable ASCII characters between ! and ~

func verifyInput(text string) bool {
	for _, c := range text {
		if c < ' ' || c > '~' {
			return false
		}
	}
	return true
}

// Add user to a channel and inform all users in the channel

func joinChannel(ch string, chMap map[string]map[*user]net.Conn, uMap map[string]*user,
	uName string, userPtr *user, conn net.Conn, s *bufio.Scanner) {
	for verifyInput(ch) == false {
		io.WriteString(conn, "Invalid characters used. Try again.\nEnter a channel: ")
		s.Scan()
		ch = s.Text()
	}

	// Check if a channel already exists

	if _, ok := chMap[ch]; !ok {
		// Channel doesn't exist; create channel

		chMap[ch] = make(map[*user]net.Conn)
		io.WriteString(conn, "\nChannel ["+ch+"] created\n")
	}

	chMap[ch][userPtr] = conn // Add user to a channel

	userPtr.cur_channel = ch // Set user's current channel

	msg := "\n(" + userPtr.nickname + ")" + " entered [" + ch + "] channel\n"
	// io.WriteString(conn, msg)
	channelMsg(ch, chMap, uMap, msg, userPtr.connection)
}

// Remove a user from a channel and inform all users in the channel

func leaveChannel(ch string, chMap map[string]map[*user]net.Conn, uMap map[string]*user,
	uName string, userPtr *user, conn net.Conn, s *bufio.Scanner) {
	delete(chMap[ch], userPtr)
	userPtr.cur_channel = ""
	msg := "\n(" + userPtr.nickname + ")" + " left [" + ch + "] channel\n"
	channelMsg(ch, chMap, uMap, msg, userPtr.connection)
}

// Check if a channel already exists

func authChannel(chMap map[string]map[*user]net.Conn, ch string) bool {
	if _, ok := chMap[ch]; ok {
		return true
	} else {
		return false
	}
}

// Add or change nickname

func nickOps(newNick string, nMap map[string]*user, userPtr *user, removeNick bool) {
	if removeNick {
		// Remove current/old nickname from nicknameMap

		delete(nMap, userPtr.nickname)

	}
	// Add new nickname to nicknameMap

	nMap[newNick] = userPtr
	userPtr.nickname = newNick
}

// Create a new user

func createUser(uName string, newUser map[string]*user) *user {
	newUser[uName] = new(user)
	newUser[uName].username = uName
	return (newUser[uName])
}

// Check if user already exists

func authUser(uMap map[string]*user, name string) bool {
	if _, ok := uMap[name]; ok {
		return true
	} else {
		return false
	}
}

// Send message to all users in a channel

func channelMsg(ch string, chMap map[string]map[*user]net.Conn, uMap map[string]*user, msg string, senderConn net.Conn) {
	for _, uConnect := range chMap[ch] {
		if uConnect != senderConn {
			io.WriteString(uConnect, "\n"+msg)
		}
	}
	for _, u := range uMap {
		if u.cur_channel == ch && u.connection != senderConn {
			msg = "[" + u.cur_channel + "] " + u.nickname + ": "
			io.WriteString(u.connection, msg)
		}
	}
}

// Send message to a specified user

func privateMsg(dst string, userPtr map[string]*user, msg string, src string) {
	if userPtr[dst].connection != nil {
		msg = msg + "[" + userPtr[dst].cur_channel + "]" + userPtr[dst].nickname + ": "
		io.WriteString(userPtr[dst].connection, msg)
	} else {
		msg = "(" + userPtr[dst].nickname + ") is not currently online.\n"
		io.WriteString(userPtr[src].connection, msg)
	}
}
