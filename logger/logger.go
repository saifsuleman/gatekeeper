package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	File *os.File
}

func (l *Logger) Write(data []byte) (n int, err error) {
	fmt.Print(string(data))
	return l.File.Write(data)
}

func InitializeLogger(filepath string) Logger {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		file, err := os.Create(filepath)
		if err != nil {
			panic(err)
		}
		file.Close()
	}

	file, err := os.OpenFile(filepath, os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	logger := Logger{file}
	log.SetOutput(&logger)
	return logger
}
