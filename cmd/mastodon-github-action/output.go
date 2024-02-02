package main

import (
	"fmt"
	"os"
)

// writeOutput writes the output to the appropriate destination based on the environment.
func writeOutput(key, value string) {
	if ghOutput, exists := os.LookupEnv("GITHUB_OUTPUT"); exists {
		f, err := os.OpenFile(ghOutput, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Println("Error opening GITHUB_OUTPUT file:", err)
			return
		}

		//nolint:errcheck
		defer f.Close()

		_, err = fmt.Fprintf(f, "%s=%s\n", key, value)
		if err != nil {
			fmt.Println("Error writing to GITHUB_OUTPUT file:", err)
		}
	} else {
		fmt.Printf("::set-output name=%s::%s\n", key, value)
	}
}

// setActionOutputs sets the GitHub Action outputs, with backward compatibility for
// self-hosted runners without a GITHUB_OUTPUT environment file.
//
//nolint:unused
func setActionOutputs(outputPairs map[string]string) {
	for key, value := range outputPairs {
		writeOutput(key, value)
	}
}

// setActionOutput sets a single GitHub Action output, with backward compatibility for
// self-hosted runners without a GITHUB_OUTPUT environment file.
func setActionOutput(outputName, value string) {
	writeOutput(outputName, value)
}
