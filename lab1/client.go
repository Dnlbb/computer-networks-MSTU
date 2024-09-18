package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type Numbers struct {
	A int `json:"a"`
	B int `json:"b"`
}

type Results struct {
	GCD int `json:"gcd"`
	LCM int `json:"lcm"`
}

func main() {

	conn, err := net.Dial("tcp", ":8081")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	var a, b int
	fmt.Print("Enter first number: ")
	fmt.Scan(&a)
	fmt.Print("Enter second number: ")
	fmt.Scan(&b)

	numbers := Numbers{A: a, B: b}
	data, err := json.Marshal(numbers)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Println("Error sending data:", err)
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	var result Results
	err = json.Unmarshal([]byte(response), &result)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Выводим результаты
	fmt.Printf("GCD: %d, LCM: %d\n", result.GCD, result.LCM)
}
