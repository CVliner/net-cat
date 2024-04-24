package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	users       = make(map[string]net.Conn)
	left        = make(chan message)
	connected   = make(chan message)
	messages    = make(chan message)
	tempHistory = []byte{}
	m           sync.Mutex
)

type message struct {
	text    string
	name    string
	time    string
	address string
}

func main() {
	port := "8989"
	if len(os.Args) > 1 {
		if isValidPort(os.Args[1]) && len(os.Args) == 2 {
			port = os.Args[1]
		} else {
			fmt.Println("[USAGE]: ./TCPChat $port")
			return
		}
	}

	fmt.Println("Server listening on " + port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	go sendMessage(messages)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		m.Lock()
		if len(users) == 10 {
			conn.Write([]byte("\nChat is full. Wait for somebody is left\n"))
			conn.Close()
		} else {
			conn.Write([]byte("WELCOME TO TCP-CHAT!\n"))
			conn.Write([]byte(printWelcome()))
			go listenConnection(conn)
		}
		m.Unlock()
	}
}

func listenConnection(conn net.Conn) {
	defer conn.Close()
	time := time.Now().Format("2020-01-20 15:48:41")
	name, msgCon, err := getName(conn, time)
	if err != nil {
		return
	}
	isName(name, conn)
	m.Lock()
	users[name] = conn
	m.Unlock()

	conn.Write(tempHistory)
	tempHistory = append(tempHistory, msgCon+"\n"...)
	ioutil.WriteFile("history.txt", tempHistory, 0777)

	for {
		conn.Write([]byte("[" + time + "]" + "[" + name + "]" + ": "))
		msg, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			left <- newMessage(name+" has left chat...", conn, name, time)
			m.Lock()
			delete(users, name)
			m.Unlock()
			log.Println(name, "disconnected")
			tempHistory = append(tempHistory, name+" has left chat...\n"...)
			ioutil.WriteFile("history.txt", tempHistory, 0o777)
			return
		} else if !isValidStr(msg) {
			continue
		} else {
			messages <- newMessage(msg, conn, name, time)

			tempHistory = append(tempHistory, "["+time+"]"+"["+name+"]"+msg...)
			ioutil.WriteFile("history.txt", tempHistory, 0o777)
		}

	}
}

func newMessage(msg string, conn net.Conn, name, time string) message {
	addr := conn.RemoteAddr().String()
	return message{
		text:    msg,
		name:    "[" + name + "]" + ": ",
		time:    "[" + time + "]",
		address: addr,
	}
}

func getName(conn net.Conn, time string) (string, string, error) {
	name := ""

	for name == "" || !isName(name, conn) || !isValidStr(name) {
		conn.Write([]byte("\n[PLEASE ENTER YOUR NICKNAME]: "))

		n, _, err := bufio.NewReader(conn).ReadLine()
		if err != nil {
			return "", "", err
		}
		if !isValidStr(string(n)) {
			conn.Write([]byte("\nIncorrect name.Please try again\n"))
		}
		name = string(n)
		if len(name) > 12 {
			conn.Write([]byte("\nLength of name is too long. Please enter the name equal or less than 12 letters\n"))

			name = ""
		}
	}

	log.Println(name + " connected")
	msg := name + " has joined chat..."
	connected <- newMessage(msg, conn, name, time)

	return strings.TrimSpace(name), msg, nil
}

func isName(name string, conn net.Conn) bool {
	for n := range users {
		if n == name {
			conn.Write([]byte("\nName is already in chat\n"))
			return false
		}
	}
	return true
}

func printWelcome() []byte {
	welcome, err := os.ReadFile("./welcome.txt")
	if err != nil {
		log.Fatal(err)
	}
	return welcome
}

func isValidPort(arg string) bool {
	if len(arg) != 4 {
		return false
	}
	for _, value := range arg {
		if value < '0' || value > '9' {
			return false
		}
	}
	return true
}

func isValidStr(str string) bool {
	for _, value := range str {
		if value != ' ' && value != '\n' {
			return true
		}
	}
	return false
}

func sendMessage(message <-chan message) {
	for {
		select {
		case msg := <-connected:
			m.Lock()
			for name, conn := range users {
				if msg.address == conn.RemoteAddr().String() {
					continue
				}
				conn.Write([]byte("\n" + msg.text + "\n"))
				conn.Write([]byte(msg.time + "[" + name + "]:"))
			}
			m.Unlock()
		case msg := <-message:
			m.Lock()
			for name, conn := range users {
				if msg.address == conn.RemoteAddr().String() {
					continue
				}

				conn.Write([]byte("\n" + msg.time + msg.name + msg.text))
				conn.Write([]byte(msg.time + "[" + name + "]:"))

			}
			m.Unlock()
		case msg := <-left:
			m.Lock()
			for name, conn := range users {
				conn.Write([]byte("\n" + msg.text + "\n"))
				conn.Write([]byte(msg.time + "[" + name + "]:"))
			}
			m.Unlock()
		}
	}
}
