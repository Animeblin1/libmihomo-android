package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/metacubex/mihomo/component/ca"
	"github.com/metacubex/mihomo/config"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/log"
)

var (
	mu      sync.Mutex
	running bool
)

type startResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

//export MihomoStart
func MihomoStart(configPathC *C.char, workDirC *C.char) *C.char {
	mu.Lock()
	defer mu.Unlock()

	configPath := C.GoString(configPathC)
	workDir := C.GoString(workDirC)

	res := startResult{}

	if running {
		executor.Shutdown()
		running = false
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		res.Error = fmt.Sprintf("mkdir workdir: %v", err)
		return jsonResult(res)
	}

	constant.SetHomeDir(workDir)
	constant.SetConfig(configPath)

	ca.ResetCertificate()

	rawCfg, err := config.UnmarshalRawConfig(readFile(configPath))
	if err != nil {
		res.Error = fmt.Sprintf("parse config: %v", err)
		return jsonResult(res)
	}

	cfg, err := config.ParseRawConfig(rawCfg)
	if err != nil {
		res.Error = fmt.Sprintf("build config: %v", err)
		return jsonResult(res)
	}

	log.SetLevel(log.WARNING)

	if err := executor.ApplyConfig(cfg, true); err != nil {
		res.Error = fmt.Sprintf("apply config: %v", err)
		return jsonResult(res)
	}

	running = true
	res.OK = true
	return jsonResult(res)
}

//export MihomoStop
func MihomoStop() {
	mu.Lock()
	defer mu.Unlock()
	if running {
		executor.Shutdown()
		running = false
	}
}

//export MihomoReload
func MihomoReload(configPathC *C.char) *C.char {
	mu.Lock()
	defer mu.Unlock()

	res := startResult{}
	if !running {
		res.Error = "not running"
		return jsonResult(res)
	}

	configPath := C.GoString(configPathC)
	rawCfg, err := config.UnmarshalRawConfig(readFile(configPath))
	if err != nil {
		res.Error = fmt.Sprintf("parse: %v", err)
		return jsonResult(res)
	}
	cfg, err := config.ParseRawConfig(rawCfg)
	if err != nil {
		res.Error = fmt.Sprintf("build: %v", err)
		return jsonResult(res)
	}
	if err := executor.ApplyConfig(cfg, false); err != nil {
		res.Error = fmt.Sprintf("apply: %v", err)
		return jsonResult(res)
	}
	res.OK = true
	return jsonResult(res)
}

//export MihomoVersion
func MihomoVersion() *C.char {
	return C.CString(constant.Version)
}

//export MihomoIsRunning
func MihomoIsRunning() C.int {
	mu.Lock()
	defer mu.Unlock()
	if running {
		return 1
	}
	return 0
}

//export MihomoFree
func MihomoFree(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func readFile(path string) []byte {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil
	}
	return data
}

func jsonResult(v interface{}) *C.char {
	b, _ := json.Marshal(v)
	return C.CString(string(b))
}

func main() {}
