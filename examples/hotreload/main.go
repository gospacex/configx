// Example: watch a config file and react to changes.
//
// Run:    go run ./examples/hotreload
// Then:   echo 'v: 99' > examples/hotreload/config.yaml
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/file"
)

func main() {
	c, err := configx.New(
		configx.WithSource(file.New("examples/hotreload/config.yaml")),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	c.OnChange(func(snap map[string]any) {
		fmt.Println("[reload] v =", snap["v"])
	})

	fmt.Println("initial v =", c.IntOr("v", 0))
	fmt.Println("edit examples/hotreload/config.yaml; press Ctrl-C to exit")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
