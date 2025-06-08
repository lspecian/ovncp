package backup

import (
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FileStorage implements file-based backup storage
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file storage backend
func NewFileStorage(basePath string) (*FileStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Store saves a backup to file storage
func (fs *FileStorage) Store(backup *BackupData, options *BackupOptions) (string, error) {
	// Generate filename
	timestamp := time.Now().Format("20060102-150405")
	safeName := strings.ReplaceAll(options.Name, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	
	var ext string
	switch options.Format {
	case BackupFormatJSON:
		ext = "json"
	case BackupFormatYAML:
		ext = "yaml"
	default:
		ext = "json"
	}

	if options.Compress {
		ext += ".gz"
	}
	if options.Encrypt {
		ext += ".enc"
	}

	filename := fmt.Sprintf("%s-%s.%s", safeName, timestamp, ext)
	filepath := filepath.Join(fs.basePath, filename)

	// Marshal data
	var data []byte
	var err error
	
	switch options.Format {
	case BackupFormatJSON:
		data, err = json.MarshalIndent(backup, "", "  ")
	case BackupFormatYAML:
		data, err = yaml.Marshal(backup)
	default:
		return "", fmt.Errorf("unsupported format: %s", options.Format)
	}
	
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup data: %w", err)
	}

	// Calculate checksum
	backup.Metadata.Checksum = calculateChecksum(data)
	backup.Metadata.Size = int64(len(data))

	// Compress if requested
	if options.Compress {
		compressed, err := fs.compressData(data)
		if err != nil {
			return "", fmt.Errorf("failed to compress data: %w", err)
		}
		backup.Statistics.UncompressedSize = int64(len(data))
		backup.Statistics.CompressedSize = int64(len(compressed))
		data = compressed
	}

	// Encrypt if requested
	if options.Encrypt {
		if options.EncryptionKey == "" {
			return "", fmt.Errorf("encryption key required for encrypted backup")
		}
		
		encrypted, err := fs.encryptData(data, options.EncryptionKey)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt data: %w", err)
		}
		data = encrypted
	}

	// Write to file
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	// Create metadata file
	metadataFile := filepath + ".meta"
	metadataData, err := json.MarshalIndent(backup.Metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	if err := ioutil.WriteFile(metadataFile, metadataData, 0644); err != nil {
		// Clean up backup file
		os.Remove(filepath)
		return "", fmt.Errorf("failed to write metadata file: %w", err)
	}

	return backup.Metadata.ID, nil
}

// Retrieve gets a backup from file storage
func (fs *FileStorage) Retrieve(backupID string) (*BackupData, error) {
	// Find backup file by ID
	files, err := fs.findBackupFiles()
	if err != nil {
		return nil, err
	}

	var backupFile string
	for _, file := range files {
		metadataFile := file + ".meta"
		if _, err := os.Stat(metadataFile); err != nil {
			continue
		}

		// Read metadata to check ID
		metadataData, err := ioutil.ReadFile(metadataFile)
		if err != nil {
			continue
		}

		var metadata BackupMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue
		}

		if metadata.ID == backupID {
			backupFile = file
			break
		}
	}

	if backupFile == "" {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}

	// Read backup file
	data, err := ioutil.ReadFile(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	// Decrypt if encrypted
	if strings.HasSuffix(backupFile, ".enc") {
		// Note: In real implementation, would need to pass decryption key
		return nil, fmt.Errorf("encrypted backups not yet supported for retrieval")
	}

	// Decompress if compressed
	if strings.Contains(backupFile, ".gz") {
		decompressed, err := fs.decompressData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
		data = decompressed
	}

	// Unmarshal data
	var backup BackupData
	if strings.Contains(backupFile, ".json") {
		if err := json.Unmarshal(data, &backup); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	} else if strings.Contains(backupFile, ".yaml") {
		if err := yaml.Unmarshal(data, &backup); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	}

	return &backup, nil
}

// List returns all available backups
func (fs *FileStorage) List() ([]*BackupMetadata, error) {
	files, err := fs.findBackupFiles()
	if err != nil {
		return nil, err
	}

	var backups []*BackupMetadata
	for _, file := range files {
		metadataFile := file + ".meta"
		if _, err := os.Stat(metadataFile); err != nil {
			continue
		}

		// Read metadata
		metadataData, err := ioutil.ReadFile(metadataFile)
		if err != nil {
			continue
		}

		var metadata BackupMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue
		}

		// Get file info for size
		info, err := os.Stat(file)
		if err == nil {
			metadata.Size = info.Size()
		}

		backups = append(backups, &metadata)
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Delete removes a backup from storage
func (fs *FileStorage) Delete(backupID string) error {
	// Find backup file by ID
	files, err := fs.findBackupFiles()
	if err != nil {
		return err
	}

	var found bool
	for _, file := range files {
		metadataFile := file + ".meta"
		if _, err := os.Stat(metadataFile); err != nil {
			continue
		}

		// Read metadata to check ID
		metadataData, err := ioutil.ReadFile(metadataFile)
		if err != nil {
			continue
		}

		var metadata BackupMetadata
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			continue
		}

		if metadata.ID == backupID {
			// Delete both files
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("failed to delete backup file: %w", err)
			}
			if err := os.Remove(metadataFile); err != nil {
				return fmt.Errorf("failed to delete metadata file: %w", err)
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	return nil
}

// Exists checks if a backup exists
func (fs *FileStorage) Exists(backupID string) (bool, error) {
	backups, err := fs.List()
	if err != nil {
		return false, err
	}

	for _, backup := range backups {
		if backup.ID == backupID {
			return true, nil
		}
	}

	return false, nil
}

// findBackupFiles finds all backup files in the storage directory
func (fs *FileStorage) findBackupFiles() ([]string, error) {
	entries, err := ioutil.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip metadata files
		if strings.HasSuffix(name, ".meta") {
			continue
		}

		// Check if it's a backup file
		if strings.Contains(name, ".json") || strings.Contains(name, ".yaml") {
			files = append(files, filepath.Join(fs.basePath, name))
		}
	}

	return files, nil
}

// compressData compresses data using gzip
func (fs *FileStorage) compressData(data []byte) ([]byte, error) {
	var buf []byte
	writer := gzip.NewWriter(&simpleWriter{&buf})
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf, nil
}

// decompressData decompresses gzip data
func (fs *FileStorage) decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(&simpleReader{data, 0})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

// encryptData encrypts data using AES-GCM
func (fs *FileStorage) encryptData(data []byte, key string) ([]byte, error) {
	// Derive key from password
	keyHash := sha256.Sum256([]byte(key))
	
	// Create cipher
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptData decrypts AES-GCM encrypted data
func (fs *FileStorage) decryptData(data []byte, key string) ([]byte, error) {
	// Derive key from password
	keyHash := sha256.Sum256([]byte(key))
	
	// Create cipher
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Helper types for compression

type simpleWriter struct {
	buf *[]byte
}

func (w *simpleWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

type simpleReader struct {
	data []byte
	pos  int
}

func (r *simpleReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}