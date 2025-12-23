package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Game     GameConfig
	Logging  LoggingConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
	Host string
	Env  string // "development" or "production"
}

// GameConfig holds game-related configuration
type GameConfig struct {
	MinPlayers            int
	MaxPlayers            int
	VotingDurationSeconds int
	RoleRevealSeconds     int
	ReconnectGracePeriod  time.Duration
	RoomCodeLength        int
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level  string
	Format string // "json" or "text"
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
			Env:  getEnv("ENV", "development"),
		},
		Game: GameConfig{
			MinPlayers:            getEnvInt("MIN_PLAYERS", 4),
			MaxPlayers:            getEnvInt("MAX_PLAYERS", 10),
			VotingDurationSeconds: getEnvInt("VOTING_DURATION_SECONDS", 20),
			RoleRevealSeconds:     getEnvInt("ROLE_REVEAL_SECONDS", 5),
			ReconnectGracePeriod:  time.Duration(getEnvInt("RECONNECT_GRACE_PERIOD_SECONDS", 120)) * time.Second,
			RoomCodeLength:        getEnvInt("ROOM_CODE_LENGTH", 6),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// GetAddr returns the server address in host:port format
func (c *Config) GetAddr() string {
	return c.Server.Host + ":" + c.Server.Port
}

// getEnv returns an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt returns an environment variable as an integer or a default value
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

