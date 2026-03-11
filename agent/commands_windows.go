package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"unicode"
)

func run(command string) (result string, err error) {
	// According to...
	// https://stackoverflow.com/questions/50809752/golang-invoking-powershell-exe-always-returns-ascii-characters
	// theres some possiblility we'd need to establish a codepage value for each command
	// command = fmt.Sprintf(`powershell -c chcp 65001 > $null; "%s")`, command)

	command = fmt.Sprintf(`powershell -c "%s")`, command)
	log.Printf("Running command \"%s\"", command)

	args, err := quotedStringSplit(command)
	if checkError(err) {
		return
	}
	// log.Printf("%+v", args)

	// var cmd *exec.Cmd
	// if len(args) == 1 {
	// 	cmd = exec.Command(args[0])
	// } else if len(args) > 1 {
	cmd := exec.Command(args[0], args[1:]...)
	// }

	out, err := cmd.CombinedOutput()
	if checkError(err) {
		log.Printf("error: Output: %s", string(out))
		return "", err
	}

	log.Printf("Output: %s", string(out))

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
