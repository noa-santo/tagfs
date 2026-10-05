package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	NamePatterns        []string `json:"name_patterns"`
	ForceMimeTypes      bool     `json:"force_mime_types"`
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
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("config contains multiple JSON values")
		}
		return fmt.Errorf("invalid trailing config data: %w", err)
	}
	if cfg.StoragePath == "" || cfg.MountPath == "" || cfg.InboxDir == "" {
		return fmt.Errorf("storage_path, mount_path, and inbox_dir are required")
	}
	storage, err := filepath.Abs(cfg.StoragePath)
	if err != nil {
		return fmt.Errorf("invalid storage_path: %w", err)
	}
	mount, err := filepath.Abs(cfg.MountPath)
	if err != nil {
		return fmt.Errorf("invalid mount_path: %w", err)
	}
	if storage == mount || isWithin(storage, mount) || isWithin(mount, storage) {
		return fmt.Errorf("mount_path and storage_path must not contain one another")
	}
	if err := os.MkdirAll(filepath.Join(storage, ".data"), 0700); err != nil {
		return fmt.Errorf("creating storage directory: %w", err)
	}
	if err := os.MkdirAll(mount, 0755); err != nil {
		return fmt.Errorf("creating mount directory: %w", err)
	}
	cfg.StoragePath, cfg.MountPath = storage, mount

	globalCfg = cfg
	loaded = true
	return nil
}

func isWithin(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()

	if !loaded {
		log.Panic("config.Get() called before config.InitConfig() was executed.")
	}

	return globalCfg
}
