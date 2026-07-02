package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

var (
	dbLogger = log.New(os.Stdout, "CONFIG: ", log.LstdFlags)

	globalCfg Config
	mu        sync.RWMutex
	loaded    bool
)

type Config struct {
	MountPath       string   `json:"mount_path"`
	StoragePath     string   `json:"storage_path"`
	PassthroughDirs []string `json:"passthrough_dirs"`
	InboxDir        string   `json:"inbox_dir"`
}

func InitConfig(path string) error {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		dbLogger.Printf("Error opening config file at %s: %v", path, err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			dbLogger.Printf("Error closing file %s: %v", path, err)
		}
	}(file)

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		dbLogger.Printf("Error decoding JSON config: %v", err)
		return err
	}

	globalCfg = cfg
	loaded = true
	return nil
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()

	if !loaded {
		log.Panic("config.Get() called before config.InitConfig() was executed.")
	}

	return globalCfg
}
