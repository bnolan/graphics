package main

import (
	"fmt"
	"os"
	"time"
)

func newLogger(file string) *logger {
	l := &logger{}
	f, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	l.file = f
	l.Println("Program started")
	return l
}

type logger struct {
	file *os.File
}

func (l *logger) close() error {
	l.Println("Program stopped")
	return l.file.Close()
}

func (l *logger) Println(a ...interface{}) {
	args := append([]interface{}{l.ts()}, a...)
	_, err := fmt.Fprintln(l.file, args...)
	if err != nil {
		panic(err)
	}
}

func (l *logger) Printf(format string, a ...interface{}) {
	args := append([]interface{}{l.ts()}, a...)
	_, err := fmt.Fprintf(l.file, "%s "+format, args...)
	if err != nil {
		panic(err)
	}
}

func (l *logger) ErrorLn(inError error) {
	_, err := fmt.Fprintf(l.file, "%s %v\n", l.ts(), inError)
	fmt.Fprintf(os.Stderr, "%s %v\n", l.ts(), inError)
	if err != nil {
		panic(err)
	}
}

func (l *logger) ts() string {
	return time.Now().Format("15:04:05.000000000")
}
