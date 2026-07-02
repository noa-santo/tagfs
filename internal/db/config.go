package db

import (
	"log"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

var (
	bucketName = []byte("config")
	dbLogger   = log.New(os.Stdout, "DB: ", log.LstdFlags)
)

// PassthroughDirs todo: replace with real config
var PassthroughDirs = []string{".config", ".passthrough"}
var InboxDir = "Inbox"

// Config holds the paths we need to persist
type Config struct {
	MountPath       string   `json:"mount_path"`
	StoragePath     string   `json:"storage_path"`
	PassthroughDirs []string `json:"passthrough_dirs"`
}

func getDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	tagfsDir := filepath.Join(configDir, "tagfs")
	err = os.MkdirAll(tagfsDir, 0755)
	if err != nil {
		return "", err
	}
	return filepath.Join(tagfsDir, "tagfs.db"), nil
}

// SaveConfig saves the paths into BoltDB
func SaveConfig(cfg Config) error {
	path, err := getDBPath()
	if err != nil {
		dbLogger.Fatalf("Error getting DB Path: %v", err)
		return err
	}
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		dbLogger.Fatalf("Error opening database: %v", err)
		return err
	}
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			dbLogger.Fatalf("Failed to close DB: %v", err)
		}
	}(db)

	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			dbLogger.Fatalf("Error creating bucket: %v", err)
			return err
		}
		err = b.Put([]byte("mount"), []byte(cfg.MountPath))
		if err != nil {
			dbLogger.Fatalf("Error saving mount path to bucket: %v", err)
			return err
		}
		err = b.Put([]byte("storage"), []byte(cfg.StoragePath))
		if err != nil {
			dbLogger.Fatalf("Error saving storage path to bucket: %v", err)
			return err
		}
		return nil
	})
}

// LoadConfig retrieves the paths from BoltDB
func LoadConfig() (Config, error) {
	path, err := getDBPath()
	if err != nil {
		dbLogger.Fatalf("Error getting DB Path: %v", err)
		return Config{}, err
	}
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		dbLogger.Fatalf("Error opening database: %v", err)
		return Config{}, err
	}
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			dbLogger.Fatalf("Failed to close DB: %v", err)
		}
	}(db)

	var cfg Config
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			dbLogger.Printf("Error opening bucket: %v", bucketName)
			return os.ErrNotExist
		}
		cfg.MountPath = string(b.Get([]byte("mount")))
		cfg.StoragePath = string(b.Get([]byte("storage")))
		return nil
	})
	return cfg, err
}
