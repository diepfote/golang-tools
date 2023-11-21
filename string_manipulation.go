package main

import "strings"

func reverse(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func stringInArray(arr []string, str string) bool {
	// prettyPrintArray("DEBUG", "arr", arr)
	// debug("stringInArray str: %v", str)
	for _, a := range arr {

		//	func HasPrefix(s, prefix string) bool
		if strings.HasPrefix(str, a) || strings.HasSuffix(str, a) {
			return true
		}
	}
	return false
}
