package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() Config {
	path, err := getConfigFilePath()
	if err != nil {
		fmt.Printf("Error getting file path %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error opening file %v", err)
		return Config{}
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Printf("Error parsing JSON %v", err)
		return Config{}
	}

	return config
}

func SetUser(config Config) error {

	jsonData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return err
	}
	return nil
}

func getConfigFilePath() (string, error) {
	path, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path += "/" + configFileName
	return path, nil
}
