package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type Numbers struct {
	A int `json:"a"`
	B int `json:"b"`
}

type Results struct {
	GCD int `json:"gcd"`
	LCM int `json:"lcm"`
}

func gcd(a, b int) int {
	if b == 0 {
		return a
	}
	return gcd(b, a%b)
}

func lcm(a, b int) int {
	return (a * b) / gcd(a, b)
}

func main() {

	fmt.Println("Launching server...")

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error setting up listener:", err)
		return
	}

	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("Error accepting connection:", err)
		return
	}

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}

		fmt.Println("Message Received:", string(message))

		message = strings.TrimSpace(message)

		var numbers Numbers
		err = json.Unmarshal([]byte(message), &numbers)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			continue
		}

		gcdResult := gcd(numbers.A, numbers.B)
		lcmResult := lcm(numbers.A, numbers.B)

		fmt.Printf("GCD: %d, LCM: %d\n", gcdResult, lcmResult)

		result := Results{
			GCD: gcdResult,
			LCM: lcmResult,
		}

		response, err := json.Marshal(result)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
			continue
		}

		conn.Write([]byte(string(response) + "\n"))
	}
}
