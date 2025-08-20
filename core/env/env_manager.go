package env

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ccagent/core/log"
	"github.com/joho/godotenv"
)

type EnvManager struct {
	mu       sync.RWMutex
	envVars  map[string]string
	envPath  string
	ticker   *time.Ticker
	stopChan chan struct{}
}

func NewEnvManager() (*EnvManager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	envPath := filepath.Join(configDir, ".env")
	
	em := &EnvManager{
		envVars:  make(map[string]string),
		envPath:  envPath,
		stopChan: make(chan struct{}),
	}

	if err := em.Load(); err != nil {
		log.Error("Failed to load initial environment variables: %v", err)
	}

	return em, nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	configDir := filepath.Join(homeDir, ".config", "ccagent")
	
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	
	return configDir, nil
}

func (em *EnvManager) Load() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, err := os.Stat(em.envPath); os.IsNotExist(err) {
		log.Debug("No .env file found at %s, using system environment variables only", em.envPath)
		return nil
	}

	envMap, err := godotenv.Read(em.envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	for key, value := range envMap {
		em.envVars[key] = value
	}

	log.Debug("Loaded %d environment variables from %s", len(envMap), em.envPath)
	return nil
}

func (em *EnvManager) Get(key string) string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if value, exists := em.envVars[key]; exists {
		return value
	}

	return os.Getenv(key)
}

func (em *EnvManager) Reload() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, err := os.Stat(em.envPath); os.IsNotExist(err) {
		log.Debug("No .env file found at %s during reload", em.envPath)
		return nil
	}

	envMap, err := godotenv.Read(em.envPath)
	if err != nil {
		return fmt.Errorf("failed to reload .env file: %w", err)
	}

	for key := range em.envVars {
		delete(em.envVars, key)
	}

	for key, value := range envMap {
		em.envVars[key] = value
	}

	log.Info("Reloaded %d environment variables from %s", len(envMap), em.envPath)
	return nil
}

func (em *EnvManager) StartPeriodicRefresh(interval time.Duration) {
	em.ticker = time.NewTicker(interval)
	
	go func() {
		log.Info("Started periodic environment variable refresh every %s", interval)
		
		for {
			select {
			case <-em.ticker.C:
				if err := em.Reload(); err != nil {
					log.Error("Failed to reload environment variables: %v", err)
				}
			case <-em.stopChan:
				log.Info("Stopping periodic environment variable refresh")
				return
			}
		}
	}()
}

func (em *EnvManager) Stop() {
	if em.ticker != nil {
		em.ticker.Stop()
	}
	
	close(em.stopChan)
}