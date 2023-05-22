package data

import (
	"errors"
	"regexp"
)

func SanitizeAnswer(ans string) (string, error) {
	re := regexp.MustCompile(`\{[^{}]*\}`)
	match := re.FindString(ans)
	if match == "" {
		return "", errors.New("error sanitizing answer")
	}
	return match, nil
}
