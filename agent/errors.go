package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func checkError(err error) bool {
	if err != nil {
		PrintErrorN(err, 1)
		// log.Print(err)
		return true
	}
	return false
}

func ErrorCaller() {
	for i := 0; i < 10; i++ {
		_, fileName, lineNum, _ := runtime.Caller(i)
		if fileName == "" {
			break
		}
		fmt.Printf("%s, line %d\n", fileName, lineNum)
	}
}

func getFrame(callerOffset int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := 4

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(callerOffset, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

func GetFileAndLine(offset int) (string, int) {
	frame := getFrame(offset)
	return frame.File, frame.Line
}

func LogError(db *sql.DB, err error) {
	var (
		app  string
		file string
		line int
	)

	e, _ := os.Executable()
	app = filepath.Base(e)
	file, line = GetFileAndLine(0)

	log.Print(err)

	if db == nil {
		log.Panicf("Database for '%s' at '%s' line %d is NIL.\nError: %s", app, file, line, err.Error())
	}

	_, err = db.ExecContext(context.Background(), "INSERT INTO errors (app, file, line, msg) VALUES (?,?,?,?)", app, file, line, err.Error())
	if err != nil {
		log.Print(err)
	}
}

func LogErrorN(db *sql.DB, err error, callerOffset int) {
	var (
		app  string
		file string
		line int
	)

	e, _ := os.Executable()
	app = filepath.Base(e)
	file, line = GetFileAndLine(callerOffset)

	if db == nil {
		log.Panicf("Database for '%s' at '%s' line %d is NIL.\nError: %s", app, file, line, err.Error())
	}

	file = filepath.Base(file)

	if len(file) > 64 {
		file = file[:64]
	}

	_, err = db.ExecContext(context.Background(), "INSERT INTO errors (app, file, line, msg) VALUES (?,?,?,?)", app, file, line, err.Error())
	if err != nil {
		log.Print(err)
	}
}

func PrintErrorN(err error, callerOffset int) {
	var (
		// app  string
		file string
		line int
	)

	file, line = GetFileAndLine(callerOffset)
	file = filepath.Base(file)

	log.Printf("%s:%d: %s\n", file, line, err.Error())
}
