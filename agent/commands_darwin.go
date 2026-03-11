package main

import (
	"context"
	"encoding/hex"
	"log"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

func runWithContext(ctx context.Context, command string) (string, error) {
	log.Printf("Running command \"%s\"", "/bin/bash -c \""+command+"\"")

	args, err := quotedStringSplit("/bin/bash -c \"" + command + "\"")
	if checkError(err) {
		return "", err
	}

	ctx, cf := context.WithTimeout(ctx, time.Minute)
	defer cf()

	cmd := &exec.Cmd{}
	if len(args) == 1 {
		cmd = exec.CommandContext(ctx, args[0])
	} else if len(args) > 1 {
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}

	out, err := cmd.CombinedOutput()
	if checkError(err) {
		log.Printf("out: %s", string(out))
		return "", err
	}

	var safedOutput strings.Builder
	for b := range out {
		if !unicode.IsPrint(rune(b)) {
			safedOutput.WriteString(hex.EncodeToString([]byte{byte(b)}))
		} else {
			safedOutput.WriteByte(byte(b))
		}
	}

	return string(out), err
}

func run(command string) (string, error) {
	// command = fmt.Sprintf(`$(/bin/bash -c 	log.Printf("Running command \"%s\"", "/bin/bash -c \""+command+"\"")

	args, err := quotedStringSplit("/bin/bash -c \"" + command + "\"")
	if checkError(err) {
		return "", err
	}
	// log.Printf("%+v", args)

	ctx, cf := context.WithTimeout(context.Background(), time.Minute)

	defer cf()

	cmd := &exec.Cmd{}
	if len(args) == 1 {
		cmd = exec.CommandContext(ctx, args[0])
	} else if len(args) > 1 {
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}

	out, err := cmd.CombinedOutput()
	if checkError(err) {
		log.Printf("out: %s", string(out))
		return "", err
	}

	// log.Printf("Output: %s", string(out))

	//err = cmd.Wait()
	//if checkError(err) {
	//	return "", err
	//}

	var safedOutput strings.Builder
	for b := range out {
		if !unicode.IsPrint(rune(b)) {
			safedOutput.WriteString(hex.EncodeToString([]byte{byte(b)}))
		} else {
			safedOutput.WriteByte(byte(b))
		}
	}

	return string(out), err
}
