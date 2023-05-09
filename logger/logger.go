package logger

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Logger struct {
	File *os.File
	CachedLog []byte
}

func (l *Logger) Write(data []byte) (n int, err error) {
	fmt.Print(string(data))
	l.CachedLog = append(l.CachedLog, data...)
	return l.File.Write(data)
}

func (l *Logger) Close() error {
	return l.File.Close()
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
	cached, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	logger := Logger{
		File: file,
		CachedLog: cached,
	}
	log.SetOutput(&logger)
	return logger
}
