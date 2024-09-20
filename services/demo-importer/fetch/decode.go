package fetch

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"fmt"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"io"
	"os"
)

func DecodeFile(password string) ([]byte, error) {
	encryptedData, err := os.ReadFile(types.DemoDataFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v\n", err)
	}

	decryptedReader, err := decryptAES256CBC(encryptedData, password)
	if err != nil {
		return nil, fmt.Errorf("error decrypting file: %v\n", err)
	}
	return decryptedReader, nil
}

func deriveKey(password string) []byte {
	hash := md5.New()
	io.WriteString(hash, password)
	return hash.Sum(nil)
}

func decryptAES256CBC(encryptedData []byte, password string) ([]byte, error) {
	iv := encryptedData[:aes.BlockSize]
	encryptedData = encryptedData[aes.BlockSize:]

	key := deriveKey(password)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	decryptedData := make([]byte, len(encryptedData))
	mode.CryptBlocks(decryptedData, encryptedData)

	decryptedData = unpad(decryptedData)

	return decryptedData, nil
}

func unpad(data []byte) []byte {
	paddingLength := int(data[len(data)-1])
	return data[:len(data)-paddingLength]
}
