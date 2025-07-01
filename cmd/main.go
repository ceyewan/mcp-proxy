package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ceyewan/mcp-proxy/internal/app"
)

var BuildVersion = "dev"

func main() {
	conf := flag.String("config", "config.json", "path to config file or a http(s) url")
	version := flag.Bool("version", false, "print version and exit")
	help := flag.Bool("help", false, "print help and exit")
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *version {
		fmt.Println(BuildVersion)
		return
	}

	// 创建应用实例
	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// 运行应用
	if err := application.Run(*conf); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
