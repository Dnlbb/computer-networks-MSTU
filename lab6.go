package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

func main() {
	conn, err := ftp.Dial("students.yss.su:21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Quit()

	err = conn.Login("login", "pass")
	if err != nil {
		log.Fatalf("Не удалось выполнить логин: %v", err)
	}

	fmt.Println("Успешное подключение и авторизация. Ожидаю команду...")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Введите команду (changeDir <путь>,\n upload <локальный файл> <удалённый файл>,\n download <удалённый файл> <локальный файл>," +
			"\n createDir <путь>, removeFile <путь>,\n listFiles <путь>, removeEmptyDir <путь>,\n removeDirRecursive <путь>, exit): ")
		scanner.Scan()
		command := scanner.Text()

		args := strings.Fields(command)

		if len(args) == 0 {
			fmt.Println("Неверная команда.")
			continue
		}

		switch args[0] {
		case "changeDir":
			if len(args) < 2 {
				fmt.Println("Укажите путь для смены директории.")
				continue
			}
			changeDir(conn, args[1])

		case "upload":
			if len(args) < 3 {
				fmt.Println("Укажите локальный файл и удалённый файл для загрузки.")
				continue
			}
			uploadFile(conn, args[1], args[2])

		case "download":
			if len(args) < 3 {
				fmt.Println("Укажите удалённый файл и локальный файл для скачивания.")
				continue
			}
			downloadFile(conn, args[1], args[2])

		case "createDir":
			if len(args) < 2 {
				fmt.Println("Укажите путь для создания директории.")
				continue
			}
			createDir(conn, args[1])

		case "removeFile":
			if len(args) < 2 {
				fmt.Println("Укажите путь к файлу для удаления.")
				continue
			}
			removeFile(conn, args[1])

		case "listFiles":
			if len(args) < 2 {
				fmt.Println("Укажите путь для получения содержимого директории.")
				continue
			}
			listFiles(conn, args[1])

		case "removeEmptyDir":
			if len(args) < 2 {
				fmt.Println("Укажите путь к пустой директории для удаления.")
				continue
			}
			removeEmptyDir(conn, args[1])

		case "removeDirRecursive":
			if len(args) < 2 {
				fmt.Println("Укажите путь к директории для рекурсивного удаления.")
				continue
			}
			removeDirRecursive(conn, args[1])

		case "exit":
			fmt.Println("Завершаю программу...")
			return

		default:
			fmt.Println("Неизвестная команда.")
		}
	}
}

func changeDir(conn *ftp.ServerConn, dir string) {
	err := conn.ChangeDir(dir)
	if err != nil {
		log.Printf("Не удалось перейти в директорию %s: %v", dir, err)
	} else {
		log.Printf("Успешно перешли в директорию: %s", dir)
	}
}

func uploadFile(conn *ftp.ServerConn, localPath, remotePath string) {
	file, err := os.Open(localPath)
	if err != nil {
		log.Printf("Не удалось открыть локальный файл %s: %v", localPath, err)
		return
	}
	defer file.Close()

	err = conn.Stor(remotePath, file)
	if err != nil {
		log.Printf("Не удалось загрузить файл на сервер: %v", err)
	} else {
		log.Printf("Файл %s успешно загружен на сервер как %s", localPath, remotePath)
	}
}

func downloadFile(conn *ftp.ServerConn, remotePath, localPath string) {
	file, err := os.Create(localPath)
	if err != nil {
		log.Printf("Не удалось создать локальный файл %s: %v", localPath, err)
		return
	}
	defer file.Close()

	resp, err := conn.Retr(remotePath)
	if err != nil {
		log.Printf("Не удалось скачать файл с сервера: %v", err)
		return
	}
	defer resp.Close()

	_, err = file.ReadFrom(resp)
	if err != nil {
		log.Printf("Не удалось записать данные в локальный файл %s: %v", localPath, err)
	} else {
		log.Printf("Файл %s успешно скачан и сохранён как %s", remotePath, localPath)
	}
}

func createDir(conn *ftp.ServerConn, dir string) {
	err := conn.MakeDir(dir)
	if err != nil {
		log.Printf("Не удалось создать директорию %s: %v", dir, err)
	} else {
		log.Printf("Директория %s успешно создана.", dir)
	}
}

func removeFile(conn *ftp.ServerConn, file string) {
	err := conn.Delete(file)
	if err != nil {
		log.Printf("Не удалось удалить файл %s: %v", file, err)
	} else {
		log.Printf("Файл %s успешно удалён.", file)
	}
}

func listFiles(conn *ftp.ServerConn, dir string) {
	entries, err := conn.List(dir)
	if err != nil {
		log.Printf("Не удалось получить содержимое директории %s: %v", dir, err)
		return
	}
	log.Printf("Содержимое директории %s:", dir)
	for _, entry := range entries {
		fmt.Println(entry.Name)
	}
}

func removeEmptyDir(conn *ftp.ServerConn, dir string) {
	err := conn.RemoveDir(dir)
	if err != nil {
		log.Printf("Не удалось удалить пустую директорию %s: %v", dir, err)
	} else {
		log.Printf("Пустая директория %s успешно удалена.", dir)
	}
}

func removeDirRecursive(conn *ftp.ServerConn, dir string) {
	entries, err := conn.List(dir)
	if err != nil {
		log.Printf("Не удалось получить содержимое директории %s: %v", dir, err)
		return
	}

	for _, entry := range entries {

		fullPath := dir + "/" + entry.Name
		if entry.Name == "." || entry.Name == ".." {
			log.Printf("Пропускаем директорию %s", fullPath)
			continue
		}

		if entry.Type == ftp.EntryTypeFile {
			err := conn.Delete(fullPath)
			if err != nil {
				log.Printf("Не удалось удалить файл %s: %v", fullPath, err)
			} else {
				log.Printf("Файл %s успешно удалён.", fullPath)
			}
		} else if entry.Type == ftp.EntryTypeFolder {
			log.Printf("Рекурсивно обрабатываю поддиректорию: %s", fullPath)
			removeDirRecursive(conn, fullPath)
		}
	}

	err = conn.RemoveDir(dir)
	if err != nil {
		log.Printf("Не удалось удалить директорию %s: %v", dir, err)
	} else {
		log.Printf("Директория %s успешно удалена.", dir)
	}
}
