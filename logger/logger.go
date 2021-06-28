package logger

import (
	"fmt"
	"log"
	"os"
)

type logWriter struct {
	file *os.File
}

func (lw *logWriter) Write(data []byte) (n int, err error) {
	fmt.Print(string(data))
	return lw.file.Write(data)
}

func newLogWriter(file *os.File) logWriter {
	return logWriter{file}
}

func InitializeLogger(filepath string) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		file, err := os.Create(filepath)
		if err != nil {
			panic(err)
		}
		file.Close()
	}

	file, err := os.OpenFile(filepath, os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	lw := newLogWriter(file)
	log.SetOutput(&lw)
}
