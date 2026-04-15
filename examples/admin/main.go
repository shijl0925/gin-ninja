// Package main runs the focused admin gin-ninja example.
package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/shijl0925/gin-ninja/bootstrap"
	"github.com/shijl0925/gin-ninja/examples/internal/fullapp"
	"github.com/shijl0925/gin-ninja/pkg/logger"
)

var fatalAdmin = func(v ...any) { log.Fatal(v...) }

func main() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fatalAdmin("resolve config path")
	}

	cfg := fullapp.MustLoadConfig(filepath.Join(filepath.Dir(file), "config.yaml"))
	log_ := bootstrap.InitLogger(&cfg.Log)
	defer logger.Sync()

	if err := fullapp.Run(*cfg, log_, fullapp.AdminOptions()); err != nil {
		fatalAdmin(err)
	}
}
