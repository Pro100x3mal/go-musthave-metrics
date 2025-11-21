package configs

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	ServerAddr      string
	LogLevel        string
	StoreInterval   time.Duration
	FileStoragePath string
	IsRestore       bool
	DatabaseDSN     string
	Key             string
	AuditFile       string
	AuditURL        string
	PrivateKeyPath  string
}

type JSONServerConfig struct {
	ServerAddr      string `json:"address"`
	LogLevel        string `json:"log_level"`
	StoreInterval   string `json:"store_interval"`
	FileStoragePath string `json:"file_storage_path"`
	IsRestore       *bool  `json:"restore"`
	DatabaseDSN     string `json:"database_dsn"`
	Key             string `json:"signing_key"`
	AuditFile       string `json:"audit_file"`
	AuditURL        string `json:"audit_url"`
	PrivateKeyPath  string `json:"crypto_key"`
}

const (
	defaultServerAddr    = "localhost:8080"
	defaultLogLevel      = "info"
	defaultStoreInterval = 300
	defaultIsRestore     = false
	defaultAuditFile     = "audit.json"
)

func GetConfig() (*ServerConfig, error) {
	var (
		storeInterval  int
		cfg            ServerConfig
		configFilePath string
	)

	cfg.ServerAddr = defaultServerAddr
	cfg.LogLevel = defaultLogLevel
	cfg.IsRestore = defaultIsRestore
	cfg.AuditFile = defaultAuditFile
	storeInterval = defaultStoreInterval

	var (
		flagServerAddr      string
		flagLogLevel        string
		flagStoreInterval   int
		flagFileStoragePath string
		flagIsRestore       bool
		flagDatabaseDSN     string
		flagKey             string
		flagAuditFile       string
		flagAuditURL        string
		flagPrivateKeyPath  string
	)

	flag.StringVar(&flagServerAddr, "a", "", "address of HTTP server")
	flag.StringVar(&flagLogLevel, "l", "", "log level")
	flag.IntVar(&flagStoreInterval, "i", -1, "store interval in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "", "path to metrics storage file")
	flag.BoolVar(&flagIsRestore, "r", false, "load metrics from file on startup")
	flag.StringVar(&flagDatabaseDSN, "d", "", "database PostgreSQL DSN")
	flag.StringVar(&flagKey, "k", "", "signing key")
	flag.StringVar(&flagAuditFile, "audit-file", "", "path to audit log file")
	flag.StringVar(&flagAuditURL, "audit-url", "", "URL for audit log server")
	flag.StringVar(&flagPrivateKeyPath, "crypto-key", "", "path to private key file")
	flag.StringVar(&configFilePath, "config", "", "path to JSON config file")
	flag.StringVar(&configFilePath, "c", "", "path to JSON config file")
	flag.Parse()

	if configFilePath == "" {
		if envConfigFilePath, ok := os.LookupEnv("CONFIG"); ok && envConfigFilePath != "" {
			configFilePath = envConfigFilePath
		}
	}

	if configFilePath != "" {
		if err := loadJSONConfig(configFilePath, &cfg, &storeInterval); err != nil {
			return nil, fmt.Errorf("failed to load JSON config: %w", err)
		}
	}

	if flagServerAddr != "" {
		cfg.ServerAddr = flagServerAddr
	}
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagStoreInterval >= 0 {
		storeInterval = flagStoreInterval
	}
	if flagFileStoragePath != "" {
		cfg.FileStoragePath = flagFileStoragePath
	}
	if flag.Lookup("r").Value.String() == "true" {
		cfg.IsRestore = flagIsRestore
	}
	if flagDatabaseDSN != "" {
		cfg.DatabaseDSN = flagDatabaseDSN
	}
	if flagKey != "" {
		cfg.Key = flagKey
	}
	if flagAuditFile != "" {
		cfg.AuditFile = flagAuditFile
	}
	if flagAuditURL != "" {
		cfg.AuditURL = flagAuditURL
	}
	if flagPrivateKeyPath != "" {
		cfg.PrivateKeyPath = flagPrivateKeyPath
	}

	if envServerAddr, ok := os.LookupEnv("ADDRESS"); ok && envServerAddr != "" {
		cfg.ServerAddr = envServerAddr
	}

	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok && envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	}

	if envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL"); ok && envStoreInterval != "" {
		var err error
		storeInterval, err = strconv.Atoi(envStoreInterval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse STORE_INTERVAL value %q to integer: %w", envStoreInterval, err)
		}
		if storeInterval < 0 {
			return nil, fmt.Errorf("STORE_INTERVAL value %q must be greater than 0", envStoreInterval)
		}
	}

	if envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && envFileStoragePath != "" {
		cfg.FileStoragePath = envFileStoragePath
	}

	if envIsRestore, ok := os.LookupEnv("RESTORE"); ok && envIsRestore != "" {
		var err error
		cfg.IsRestore, err = strconv.ParseBool(envIsRestore)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RESTORE value %q to boolean: %w", envIsRestore, err)
		}
	}

	if envDatabaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok && envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	if envKey, ok := os.LookupEnv("KEY"); ok && envKey != "" {
		cfg.Key = envKey
	}

	if envAuditFile, ok := os.LookupEnv("AUDIT_FILE"); ok && envAuditFile != "" {
		cfg.AuditFile = envAuditFile
	}

	if envAuditURL, ok := os.LookupEnv("AUDIT_URL"); ok && envAuditURL != "" {
		cfg.AuditURL = envAuditURL
	}

	if envPrivateKeyPath, ok := os.LookupEnv("CRYPTO_KEY"); ok && envPrivateKeyPath != "" {
		cfg.PrivateKeyPath = envPrivateKeyPath
	}

	cfg.StoreInterval = time.Duration(storeInterval) * time.Second

	return &cfg, nil
}

func loadJSONConfig(path string, cfg *ServerConfig, storeInterval *int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var jsonCfg JSONServerConfig
	if err = json.Unmarshal(data, &jsonCfg); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	if jsonCfg.ServerAddr != "" {
		cfg.ServerAddr = jsonCfg.ServerAddr
	}
	if jsonCfg.LogLevel != "" {
		cfg.LogLevel = jsonCfg.LogLevel
	}
	if jsonCfg.IsRestore != nil {
		cfg.IsRestore = *jsonCfg.IsRestore
	}
	if jsonCfg.StoreInterval != "" {
		duration, err := time.ParseDuration(jsonCfg.StoreInterval)
		if err != nil {
			return fmt.Errorf("failed to parse store_interval: %w", err)
		}
		cfg.StoreInterval = duration
		*storeInterval = int(duration.Seconds())
	}
	if jsonCfg.FileStoragePath != "" {
		cfg.FileStoragePath = jsonCfg.FileStoragePath
	}
	if jsonCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = jsonCfg.DatabaseDSN
	}
	if jsonCfg.PrivateKeyPath != "" {
		cfg.PrivateKeyPath = jsonCfg.PrivateKeyPath
	}
	if jsonCfg.Key != "" {
		cfg.Key = jsonCfg.Key
	}
	if jsonCfg.AuditFile != "" {
		cfg.AuditFile = jsonCfg.AuditFile
	}
	if jsonCfg.AuditURL != "" {
		cfg.AuditURL = jsonCfg.AuditURL
	}

	return nil
}
