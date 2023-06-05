package util

import "math/rand"

func GenerateRandomString(length int, random *rand.Rand) string {
	randStrBytes := make([]byte, length)
	shiftToSkipSymbols := 6

	for i := 0; i < length; i++ {
		symbolCodeLimiter := 'z' - 'A' - shiftToSkipSymbols
		symbolCode := random.Intn(symbolCodeLimiter)
		if symbolCode > 'Z'-'A' {
			symbolCode += shiftToSkipSymbols
		}
		randStrBytes[i] = byte('A' + symbolCode)
	}

	return string(randStrBytes)
}