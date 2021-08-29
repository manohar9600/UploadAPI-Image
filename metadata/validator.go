package metadata

import (
	"errors"
	"strings"
)

func ValidateInput(inputData InputData) error {
	if len(strings.Trim(inputData.User, " ")) == 0 {
		return errors.New("username is empty")
	}
	if len(strings.Trim(inputData.Data, " ")) == 0 {
		return errors.New("data is empty")
	}
	return nil
}
