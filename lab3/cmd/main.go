package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileInfo struct {
	Name         string
	LastModified time.Time
	Content      []byte
}

func watchDirectory(directory string, events chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				events <- event.Name
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func getLatestFileInfo(filePath string) (FileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return FileInfo{}, err
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:         filepath.Base(filePath),
		LastModified: info.ModTime(),
		Content:      content,
	}, nil
}

func retryDial(address string, maxRetries int, baseDelay time.Duration) (net.Conn, error) {
	var conn net.Conn
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = net.Dial("tcp", address)
		if err == nil {
			return conn, nil
		}
		delay := time.Duration(math.Pow(2, float64(i))) * baseDelay
		log.Printf("Не удалось подключиться к %s, попытка %d, повтор через %v", address, i+1, delay)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("не удалось подключиться к %s после %d попыток", address, maxRetries)
}

func syncFilesWithPeer(peerAddress string, fileInfo FileInfo) error {
	conn, err := retryDial(peerAddress, 5, time.Second*2)
	if err != nil {
		log.Println("Ошибка подключения к узлу:", err)
		return err
	}
	defer conn.Close()

	fmt.Fprintf(conn, "%s %d\n", fileInfo.Name, fileInfo.LastModified.Unix())
	conn.Write(fileInfo.Content)
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var fileName string
	var timestamp int64
	fmt.Fscanf(conn, "%s %d\n", &fileName, &timestamp)

	fileContent := make([]byte, 1024)
	n, _ := conn.Read(fileContent)

	localPath := filepath.Join("./local", fileName)
	localInfo, err := os.Stat(localPath)
	if os.IsNotExist(err) || localInfo.ModTime().Unix() < timestamp {
		os.WriteFile(localPath, fileContent[:n], 0644)
		log.Printf("Файл %s обновлен.", fileName)
	}
}

func main() {
	localDirectory := "./local"
	peerAddress := "185.102.139.169:8080"

	events := make(chan string)
	go watchDirectory(localDirectory, events)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Ошибка при принятии подключения:", err)
				continue
			}
			go handleConnection(conn)
		}
	}()

	for {
		select {
		case file := <-events:
			fileInfo, err := getLatestFileInfo(file)
			if err != nil {
				log.Println("Ошибка получения информации о файле:", err)
				continue
			}
			err = syncFilesWithPeer(peerAddress, fileInfo)
			if err != nil {
				log.Println("Не удалось синхронизировать файл с пиром:", err)
			}
		}
	}
}
