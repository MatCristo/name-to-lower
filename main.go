package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func GetJPGConverterPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório home: %w", err)
	}

	jpgConverterPath := filepath.Join(homeDir, "Documents", "JPGconverter")

	if _, err := os.Stat(jpgConverterPath); os.IsNotExist(err) {
		err := os.MkdirAll(jpgConverterPath, 0755)
		if err != nil {
			return "", fmt.Errorf("erro ao criar diretório %s: %w", jpgConverterPath, err)
		}
		fmt.Printf("Pasta criada: %s\n", jpgConverterPath)
	} else if err != nil {
		return "", fmt.Errorf("erro ao verificar diretório %s: %w", jpgConverterPath, err)
	}

	return jpgConverterPath, nil
}

func CreateJPGConverterDir() {
	path, err := GetJPGConverterPath()
	if err != nil {
		fmt.Printf("Erro ao criar/verificar pasta JPGconverter: %v\n", err)
		return
	}
	fmt.Printf("Pasta JPGconverter disponível em: %s\n", path)
}

func main() {
	watchDir, err := GetJPGConverterPath()
	if err != nil {
		fmt.Printf("Erro: %v\n", err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if err := watcher.Add(watchDir); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Monitorando a pasta:", watchDir)

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					oldPath := event.Name

					if waitForFileReady(oldPath) {
						dir := filepath.Dir(oldPath)
						base := filepath.Base(oldPath)
						lowerName := strings.ToLower(base)
						newPath := filepath.Join(dir, lowerName)

						if oldPath != newPath {
							err := os.Rename(oldPath, newPath)
							if err != nil {
								fmt.Printf("Erro ao renomear %s: %v\n", base, err)
							} else {
								fmt.Printf("Arquivo renomeado: %s -> %s\n", base, lowerName)
							}
						}
					} else {
						fmt.Printf("Não foi possível acessar o arquivo: %s\n", oldPath)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("Erro:", err)
			}
		}
	}()

	<-done
}

func waitForFileReady(path string) bool {
	const (
		maxTries = 30
		delay    = 200 * time.Millisecond
	)

	var lastSize int64 = -1
	for i := 0; i < maxTries; i++ {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		if info.IsDir() {
			return false
		}

		if info.Size() == lastSize {
			f, err := os.OpenFile(path, os.O_RDWR, 0)
			if err == nil {
				f.Close()
				return true
			}
		}
		lastSize = info.Size()
		time.Sleep(delay)
	}
	return false
}
