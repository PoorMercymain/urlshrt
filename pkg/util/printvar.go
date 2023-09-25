package util

import "fmt"

func PrintVariable(variable string, shortDescription string) {
	if variable != "" {
		fmt.Println("Build", shortDescription + ":", variable)
	} else {
		fmt.Println("Build", shortDescription + ": N/A")
	}
}