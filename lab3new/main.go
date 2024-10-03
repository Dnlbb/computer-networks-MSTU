package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileState struct {
	Name    string
	ModTime time.Time
}

type Peer struct {
	Addr       string
	NextPeer   string
	DirPath    string
	FileStates map[string]FileState
}

type CreateFileRequest struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

type SyncRequest struct {
	FileName  string
	Content   []byte
	ModTime   string
	PeersLeft int
}

var mu sync.Mutex
var peer *Peer

func watchDirectory(peer *Peer) {
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()

	_ = watcher.Add(peer.DirPath)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				fileInfo, _ := os.Stat(event.Name)
				mu.Lock()
				peer.FileStates[event.Name] = FileState{
					Name:    fileInfo.Name(),
					ModTime: fileInfo.ModTime(),
				}
				mu.Unlock()

				syncWithNextPeer(peer, 3)
			}
		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func syncWithNextPeer(peer *Peer, peersLeft int) {
	if peersLeft == 0 {
		return
	}

	for _, fileState := range peer.FileStates {

		url := "http://" + peer.NextPeer + "/sync"

		fileContent, _ := os.ReadFile(peer.DirPath + "/" + fileState.Name)

		reqBody := &bytes.Buffer{}
		writer := multipart.NewWriter(reqBody)

		fileWriter, _ := writer.CreateFormFile("file", fileState.Name)
		fileWriter.Write(fileContent)

		writer.WriteField("mod_time", fileState.ModTime.Format(time.RFC3339))
		writer.WriteField("peers_left", strconv.Itoa(peersLeft))
		writer.Close()

		_, err := http.Post(url, writer.FormDataContentType(), reqBody)
		if err != nil {
			log.Println("Error sending file:", err)
		}
	}
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	file, header, _ := r.FormFile("file")
	modTimeStr := r.FormValue("mod_time")
	modTime, _ := time.Parse(time.RFC3339, modTimeStr)

	peersLeftStr := r.FormValue("peers_left")
	peersLeft, _ := strconv.Atoi(peersLeftStr)
	peersLeft--

	fileName := header.Filename
	localFileState, exists := peer.FileStates[fileName]

	if !exists || localFileState.ModTime.Before(modTime) {
		out, _ := os.Create(peer.DirPath + "/" + fileName)
		io.Copy(out, file)

		peer.FileStates[fileName] = FileState{
			Name:    fileName,
			ModTime: modTime,
		}
		fmt.Println("Updated file:", fileName)
	}

	if peersLeft > 0 {
		syncWithNextPeer(peer, peersLeft)
	}
}

func handleCreateFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req CreateFileRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Ошибка при разборе JSON: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	filePath := peer.DirPath + "/" + req.FileName

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Ошибка при создании файла: %v", err)
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = file.WriteString(req.Content)
	if err != nil {
		log.Printf("Ошибка при записи в файл: %v", err)
		http.Error(w, "Unable to write to file", http.StatusInternalServerError)
		return
	}
	mu.Lock()
	peer.FileStates[req.FileName] = FileState{
		Name:    req.FileName,
		ModTime: time.Now(),
	}
	mu.Unlock()

	syncWithNextPeer(peer, 3)

	fmt.Fprintf(w, "File %s created successfully", req.FileName)
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	var files []FileState
	for _, file := range peer.FileStates {
		files = append(files, file)
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(files)
	if err != nil {
		http.Error(w, "Ошибка при кодировании JSON", http.StatusInternalServerError)
		return
	}
}

func main() {

	var addr, nextPeer, dirPath string
	flag.StringVar(&addr, "addr", "localhost:8081", "IP:port текущего пира")
	flag.StringVar(&nextPeer, "next", "localhost:8082", "IP:port следующего пира")
	flag.StringVar(&dirPath, "dir", "./dir1", "Путь к директории для синхронизации")
	flag.Parse()
	peer = &Peer{
		Addr:       addr,
		NextPeer:   nextPeer,
		DirPath:    dirPath,
		FileStates: make(map[string]FileState),
	}
	peer.FileStates["firts.txt"] = FileState{Name: "firts.txt", ModTime: time.Now()}

	http.HandleFunc("/sync", handleSync)
	http.HandleFunc("/create_file", handleCreateFile)
	http.HandleFunc("/list_files", handleListFiles)

	go watchDirectory(peer)

	log.Println("Starting peer at", addr)
	log.Fatal(http.ListenAndServe(peer.Addr, nil))
}
