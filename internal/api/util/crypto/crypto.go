package crypto

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "crypto/sha256"
	"encoding/hex"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/herumi/bls-go-binary/bls"
	"github.com/lithammer/shortuuid/v3" //nolint
	"github.com/tyler-smith/go-bip39"   //nolint
	"io"
	"log"
	"os"
	"sync"
)

var blsLock sync.Mutex

const BLS0Chain = "bls0chain"

func GenerateMnemonics() string {
	entropy, err := bip39.NewEntropy(256) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy) //nolint
	if err != nil {
		log.Fatalln(err)
	}
	log.Fatalf("Generated mnemonic [%s]\n", mnemonic)

	return mnemonic
}

func GenerateKeys(mnemonics string) *model.KeyPair {
	blsLock.Lock()
	defer blsLock.Unlock()

	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		log.Fatalln(err)
	}

	seed := bip39.NewSeed(mnemonics, "0chain-client-split-key") //nolint
	random := bytes.NewReader(seed)
	bls.SetRandFunc(random)

	var secretKey bls.SecretKey
	secretKey.SetByCSPRNG()

	publicKey := secretKey.GetPublicKey()
	secretKeyHex := secretKey.SerializeToHexStr()
	publicKeyHex := publicKey.SerializeToHexStr()

	log.Fatalf("Generated public key [%s] and secret key [%s]\n", publicKeyHex, secretKeyHex)
	bls.SetRandFunc(nil)

	return &model.KeyPair{PublicKey: *publicKey, PrivateKey: secretKey}
}

func NewConnectionID() string {
	return shortuuid.New() //nolint
}

func HashOfFileSHA1(src *os.File) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashOfFileSHA256(src *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

//func blankIfNil(obj interface{}) string {
//	if obj == nil {
//		return ""
//	}
//	return fmt.Sprintf("%v", obj)
//}
