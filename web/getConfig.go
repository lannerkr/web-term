package web

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
)

type Group struct {
	Groups []Groups
}
type Groups struct {
	Gname    string
	Sessions []Session
}
type Session struct {
	Id      SessionID
	Name    string
	Host    string
	Port    string
	User    string
	Pwd     []byte
	KeyPath string
	Usepwd  bool
}
type SessionID struct {
	Gid int
	Sid int
}

var sessionSEED []byte = []byte("a very very very very secret key")

func getConfig(result *Group) {
	// jsonFile, err := os.Open("/usr/local/witty/config/sessionconf.json")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println("seccessfully opened sessionconfig.json")
	// defer jsonFile.Close()

	byteValue, _ := os.ReadFile("/usr/local/witty/config/sessionconf.json")
	//fmt.Println(byteValue)

	//var resultn map[string]interface{}
	json.Unmarshal(byteValue, &result)

	//fmt.Println(result)
}

func encrypt(text []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionSEED)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(text []byte) ([]byte, error) {
	block, err := aes.NewCipher(sessionSEED)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}
