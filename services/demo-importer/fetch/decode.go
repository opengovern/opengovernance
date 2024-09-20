package fetch

import (
	"fmt"
	"github.com/Luzifer/go-openssl/v4"
	"os"
)

func DecryptString(filePath, passphrase string) ([]byte, error) {
	encryptedBase64String, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v\n", err)
	}

	o := openssl.New()

	dec, err := o.DecryptBytes(passphrase, []byte(encryptedBase64String), openssl.BytesToKeyMD5)
	if err != nil {
		fmt.Printf("An error occurred: %s\n", err)
	}
	return dec, nil
}
