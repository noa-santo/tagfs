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

type Rules struct {
	MimeTypes           []string `json:"mime_types"`
	ForceMimeTypes      bool     `json:"force_mime_types"`
	NamePatterns        []string `json:"name_patterns"`
	ForceNamePattern    bool     `json:"force_name_pattern"`
	AllowSubdirCreation bool     `json:"allow_subdir_creation"`
	AllowFileCreation   bool     `json:"allow_file_creation"`
}

type DirectoryConfig struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Tags           []string          `json:"tags"`
	Rules          Rules             `json:"rules"`
	Subdirectories []DirectoryConfig `json:"subdirectories,omitempty"`
	Volatile       bool              `json:"dangerous_volatile"`
}

type Config struct {
	MountPath       string            `json:"mount_path"`
	StoragePath     string            `json:"storage_path"`
	PassthroughDirs []string          `json:"passthrough_dirs"`
	InboxDir        string            `json:"inbox_dir"`
	Directories     []DirectoryConfig `json:"directories"`
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
