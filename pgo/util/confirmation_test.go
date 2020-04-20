package util

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	"bytes"
	"testing"
)

func TestAskForConfirmation(t *testing.T) {
	// returns false when there is no input
	{
		var out bytes.Buffer
		var in = bytes.NewBufferString("")

		result := askForConfirmation(in, &out, "p?")
		expected := "p?"

		if out.String() != expected {
			t.Fatalf("expected one prompt %q, got %q", expected, out.String())
		}
		if result {
			t.Fatalf("expected false, got true")
		}
	}

	// eventually returns true for "yes"
	{
		var out bytes.Buffer
		var in = bytes.NewBufferString("\na\nYeS")

		result := askForConfirmation(in, &out, "zzz")
		expected := "zzzPlease type yes or no and then press enter:\n"
		expected += "zzzPlease type yes or no and then press enter:\n"
		expected += "zzz"

		if out.String() != expected {
			t.Fatalf("expected three prompts:\n%s"+"\ngot:\n%s", expected, out.String())
		}
		if !result {
			t.Fatalf("expected true, got false")
		}
	}

	// eventually returns false for "no"
	{
		var out bytes.Buffer
		var in = bytes.NewBufferString("x\nN")

		result := askForConfirmation(in, &out, "=")
		expected := "=Please type yes or no and then press enter:\n"
		expected += "="

		if out.String() != expected {
			t.Fatalf("expected one prompt:\n%s"+"\ngot:\n%s", expected, out.String())
		}
		if result {
			t.Fatalf("expected false, got true")
		}
	}

	// returns false after a few prompts
	{
		var out bytes.Buffer
		var in = bytes.NewBufferString("wat\nhow\nto\nexit\nvim\n")

		result := askForConfirmation(in, &out, "@")
		expected := "@Please type yes or no and then press enter:\n"
		expected += "@Please type yes or no and then press enter:\n"
		expected += "@Please type yes or no and then press enter:\n"
		expected += "@Please type yes or no and then press enter:\n"
		expected += "@Assuming no...\n"

		if out.String() != expected {
			t.Fatalf("expected five prompts:\n%s"+"\ngot:\n%s", expected, out.String())
		}
		if result {
			t.Fatalf("expected false, got true")
		}
	}
}
