package util

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// AskForConfirmation uses Scanln to parse user input. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user. Typically, you should use fmt to print out a question
// before calling AskForConfirmation. E.g. fmt.Println("WARNING: Are you sure? (yes/no)")
func AskForConfirmation(NoPrompt bool, msg string) bool {
	prompt := "WARNING - " + msg + " (yes/no): "
	if msg == "" {
		prompt = "WARNING: Are you sure? (yes/no): "
	}

	return NoPrompt || askForConfirmation(os.Stdin, os.Stdout, prompt)
}

func askForConfirmation(in io.Reader, out io.Writer, prompt string) bool {
	var response string

	// ask for input a few times then give up
	remaining := 5
	for {
		fmt.Fprint(out, prompt)
		if _, err := fmt.Fscanln(in, &response); err == io.EOF {
			return false
		}
		response = strings.ToLower(response)

		if response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}

		if remaining--; remaining < 1 {
			break
		}
		fmt.Fprintln(out, "Please type yes or no and then press enter:")
	}

	fmt.Fprintln(out, "Assuming no...")
	return false
}
