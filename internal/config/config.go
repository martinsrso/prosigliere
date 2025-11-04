package config

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	Port         int    `env:"PORT" default:"8080"`
	AuthUsername string `env:"AUTH_USERNAME" default:"admin"`
	AuthPassword string `env:"AUTH_PASSWORD" default:"secret"`
	DatabaseURL  string `env:"DATABASE_URL" default:"postgres://prosig:prosig_pass@localhost:5432/prosig?sslmode=disable"`
}

func LoadConfig() (*Config, error) {
	config := &Config{}
	
	if err := loadEnvFile(".env"); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	if err := loadConfigFromEnv(config); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	return config, nil
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		value = strings.Trim(value, `"`)
		value = strings.Trim(value, `'`)

		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func loadConfigFromEnv(config interface{}) error {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		defaultValue := field.Tag.Get("default")
		envValue := os.Getenv(envTag)

		if envValue == "" {
			envValue = defaultValue
		}

		if envValue == "" {
			continue
		}

		if err := setFieldValue(fieldValue, envValue); err != nil {
			return fmt.Errorf("error setting field %s: %w", field.Name, err)
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %w", err)
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %w", err)
		}
		field.SetUint(uintValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		field.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %w", err)
		}
		field.SetFloat(floatValue)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
