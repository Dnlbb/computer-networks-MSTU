package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	host := "host"
	port := "2122"

	address := host + ":" + port

	login := "root"
	pass := "password"

	// создание конфига для клиента
	config := &ssh.ClientConfig{
		User: login,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		BannerCallback:  ssh.BannerDisplayStderr(),
		Timeout:         5 * time.Second,
	}

	// стучимся на сервер, создаем клиент
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		log.Fatal("Failed to dial: ", err)
	}
	defer client.Close()

	for {
		s, err := client.NewSession()
		if err != nil {
			log.Fatal("Failed to create a session: ", err)
		}

		fmt.Print("Enter command (or 'exit' to quit): ")
		cmd := readInput()
		if strings.ToLower(cmd) == "exit" {
			fmt.Println("Terminal closed!")
			break
		}
		var stdout bytes.Buffer
		s.Stdout = &stdout

		err = s.Run(cmd)
		if err != nil {
			fmt.Println("Error: ", err)
		} else if len(stdout.String()) != 0 {
			fmt.Print(stdout.String())
		}

		s.Close()
	}
}

func readInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return ""
	}
	return strings.TrimSpace(input)
}
