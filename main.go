package main

import (
	"fmt"

	"github.com/mackeper/m_backuper/config"
)

func main() {
	fmt.Println("Hello, World!")

	config := config.NewConfig()
	fmt.Println("Config created:", config)
}
