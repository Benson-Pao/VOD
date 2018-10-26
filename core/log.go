package core

import (
	"log"
	"os"
)

func WriteLine(FilePath string, Message string) (int, error) {

	file, err := os.OpenFile(FilePath, os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("failed opening to file:", FilePath, err)
		return 0, err
	}
	defer file.Close()

	len, err := file.WriteString(Message + "\r\n")
	if err != nil {
		log.Println("failed writing to file:", FilePath, err)
		return 0, err
	}
	return len, nil
}
