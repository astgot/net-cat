package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	maxUsers = 10
)

var (
	usersSessions = make(map[net.Conn]string) // list of users in the server
	chatUsers     = make(chan net.Conn)       // channel for added Users in the chat
	newSession    = make(chan net.Conn)       // channel for newbies
	messagesCh    = make(chan string)         // to send message everyone
	allMessages   = []string{}                // store all messages
	logoutUsers   = make(chan net.Conn)       //
	authorMessage = make(map[net.Conn]string) // to prevent sending message to author
	mutex         = &sync.Mutex{}
)

// LogIntoFile ...
func LogIntoFile() {
	file, err := os.OpenFile("net-cat_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
}

// HandleClients ...
func HandleClients() {
	// add client to channel
	for {
		select {
		// if client is newbie .... (assign his value to var session to free up the channel)
		case session := <-newSession:
			go func(session net.Conn) {
				// Output content in user console
				io.WriteString(session, WelcomePage())
				var name string
				// To prevent using empty username
				for name == "" || name == "\r" || name == "\n" {
					io.WriteString(session, "[ENTER YOUR NAME]: ")
					name, _ = bufio.NewReader(session).ReadString('\n')
					name = strings.Trim(name, "\r\n") // cut newline from name
				}
				// Is username unique in the chat?
				var isUnique bool = true
				for {
					for _, username := range usersSessions {
						if username == name {
							io.WriteString(session, "That username is busy, type another")
							io.WriteString(session, "\n[ENTER YOUR NAME]: ")
							name, _ = bufio.NewReader(session).ReadString('\n')
							name = strings.Trim(name, "\r\n")
							isUnique = false
						}
					}
					if isUnique {
						break
					}
					isUnique = true
				}
				// Output old messages
				for _, message := range allMessages {
					io.WriteString(session, message)
				}
				usersSessions[session] = name // Add the user to the map
				if len(usersSessions) >= 1 {
					messagesCh <- "\n" + name + " has joined to our chat..."
					authorMessage[session] = "\n" + name + " has joined to our chat..."
				}
				chatUsers <- session // Add the user to the channel
				log.Printf("%s has joined to TCP chat", name)

			}(session)
		// users in the chat ...
		case session := <-chatUsers:

			go func(session net.Conn, username string) {
				reader := bufio.NewReader(session)
				for {
					id := "[" + time.Now().Format("2006-01-02 15:04:05") + "]" + "[" + usersSessions[session] + "]:"
					io.WriteString(session, id)
					message, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					if message != "" && message != "\r" && message != "\n" {
						message = strings.Trim(message, "\r\n")
						messagesCh <- "\n" + id + message                  // to send everyone
						allMessages = append(allMessages, id+message+"\n") // to output to newbies
						authorMessage[session] = "\n" + id + message
						log.Printf(username + " wrote the message: " + "\"" + message + "\"")
						// LogIntoFile(username + " write message: " + "\"" + message + "\"")
					}

				}
				logoutUsers <- session // if user quit from chat
				messagesCh <- fmt.Sprintf("\n%s has left the chat ...", username)
				log.Printf("%s has left the chat ...", username)

			}(session, usersSessions[session])

		// Output messages
		case message := <-messagesCh:
			for session := range usersSessions {
				go func(user net.Conn, message string) {
					mutex.Lock()
					if authorMessage[user] != message {
						_, err := io.WriteString(user, message+"\n"+"["+time.Now().Format("2006-01-02 15:04:05")+"]"+"["+usersSessions[user]+"]:")
						if err != nil {
							logoutUsers <- user
							messagesCh <- fmt.Sprintf("%s has left the chat ...", usersSessions[user])
							log.Printf("%s has left the chat ...", usersSessions[user])
						}
					} else {
						delete(authorMessage, user)
					}
					mutex.Unlock()

				}(session, message)
			}

		// Delete session from map
		case out := <-logoutUsers:
			go func(cl net.Conn) {
				// LogIntoFile(usersSessions[cl] + " has left the chat ...")
				delete(usersSessions, cl)
			}(out)
		}
	}
}

// WelcomePage ...
func WelcomePage() string {
	linuxLogo := "         _nnnn_\n        dGGGGMMb\n       @p~qp~~qMb\n       M|@||@ M|\n       @,----.JM|\n      JS^\\__/  qKL\n     dZP        qKRb\n    dZP          qKKb\n   fZP            SMMb\n   HZM            MMMM\n   FqM            MMMM\n __| \".        |\\dS\"qML\n |    `.       | `' \\Zq\n_)      \\.___.,|     .'\n\\____   )MMMMMP|   .'\n     `-'       `--'\"\n"
	return fmt.Sprintf("Welcome to TCP-Chat!\n" + linuxLogo)
}
func main() {

	args := os.Args[1:]

	if len(args) == 0 {
		args = append(args, "8989") //default port
	} else if len(args) != 1 {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}
	port := ":" + args[0]

	// Initializing TCP server
	server, err := net.Listen("tcp", port)
	// handling validity of the port
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	LogIntoFile()
	log.Printf("Starting of TCP Chat ....")
	log.Printf("Listening on port %s ....", strings.Trim(port, ":"))
	fmt.Printf("Listening on port %s ....", strings.Trim(port, ":"))
	defer server.Close()

	// infinite cycle for accepting connections
	go func() {
		for {
			client, err := server.Accept()
			if err != nil {
				fmt.Println(err.Error(), "for cycle main func()")
				return
			}
			// Check number of users
			if len(usersSessions) >= maxUsers {
				io.WriteString(client, "Ooibai ... Oryn joq =(")
				return // need to check
			}
			newSession <- client
		}
	}()
	HandleClients()
}
