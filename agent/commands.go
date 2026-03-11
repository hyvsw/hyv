package main

import (
	"github.com/google/shlex"
)

type Command struct {
	UUID    string
	Name    string
	Input   string
	Output  string
	Special specialCommand
}

type specialCommand int64

const (
	specialCmdUpgrade specialCommand = 1 << iota
)

// ParseCommand splits a command string into tokens, preserving quoted substrings.
func quotedStringSplit(input string) ([]string, error) {
	return shlex.Split(input)
	// var tokens []string
	// var currentToken strings.Builder
	// var inQuotes bool
	// var quoteChar rune
	//
	// for i, char := range input {
	// 	switch char {
	// 	case '"', '\'':
	// 		if inQuotes {
	// 			// End quote if it matches the current quote character
	// 			if char == quoteChar {
	// 				inQuotes = false
	// 				tokens = append(tokens, currentToken.String())
	// 				currentToken.Reset()
	// 			} else {
	// 				// Append mismatched quotes as part of the token
	// 				currentToken.WriteRune(char)
	// 			}
	// 		} else {
	// 			// Start a quoted string
	// 			inQuotes = true
	// 			quoteChar = char
	// 		}
	// 	case ' ':
	// 		if inQuotes {
	// 			// Treat spaces inside quotes as part of the token
	// 			currentToken.WriteRune(char)
	// 		} else {
	// 			// End the current token and start a new one
	// 			if currentToken.Len() > 0 {
	// 				tokens = append(tokens, currentToken.String())
	// 				currentToken.Reset()
	// 			}
	// 		}
	// 	default:
	// 		// Append regular characters to the current token
	// 		currentToken.WriteRune(char)
	// 	}
	//
	// 	// Handle the last token if we're at the end of the input
	// 	if i == len(input)-1 && currentToken.Len() > 0 {
	// 		if inQuotes {
	// 			return nil, fmt.Errorf("unterminated quote detected")
	// 		}
	// 		tokens = append(tokens, currentToken.String())
	// 	}
	// }
	//
	// return tokens, nil
}
