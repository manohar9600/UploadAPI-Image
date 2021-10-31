package validators

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func ValidateInput(user string, data string) error {
	if len(strings.Trim(user, " ")) == 0 {
		return errors.New("username is empty")
	}
	if len(strings.Trim(data, " ")) == 0 {
		return errors.New("data is empty")
	}
	return nil
}

func ValidateData(data []byte, hash string) error {
	hashBytes := sha256.Sum256(data)
	computedHash := hex.EncodeToString(hashBytes[:])
	if computedHash == hash {
		return nil
	} else {
		return errors.New("hash not matched")
	}
}
