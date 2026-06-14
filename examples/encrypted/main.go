// Example: store secrets as enc(...) values; configx decrypts on load.
//
// In a real deployment the key would come from a KMS or env var. Here we
// hardcode it for demonstration only.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/encrypt"
	"github.com/gospacex/configx/source/file"
)

func main() {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	cipher, err := encrypt.NewAESGCM(key)
	if err != nil {
		log.Fatal(err)
	}

	// Generate config.yaml with an encrypted password the first time.
	cfgPath := "examples/encrypted/config.yaml"
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		ct, _ := cipher.Encrypt("super-secret")
		body := "db:\n  user: app\n  password: \"" + encrypt.Wrap(ct) + "\"\n"
		_ = os.WriteFile(cfgPath, []byte(body), 0o644)
		fmt.Println("wrote", cfgPath)
	}

	c, err := configx.New(
		configx.WithSource(file.New(cfgPath, file.WithWatch(false))),
		configx.WithCipher(cipher),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("db.user     =", c.StringOr("db.user", ""))
	fmt.Println("db.password =", c.StringOr("db.password", "")) // decrypted
}
