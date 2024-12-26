package models

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

// Config хранит конфигурацию приложения
type ConfigStruct struct {
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

var (
	DB     *pgxpool.Pool
	Config ConfigStruct
)

// InitDB инициализирует подключение к базе данных
func InitDB() {
	// Загрузка переменных окружения из .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Не удалось загрузить .env файл, используем системные переменные")
	}

	// Инициализация конфигурации
	Config = ConfigStruct{
		Port:       getEnv("PORT", "8080"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "chat_app"),
	}

	// Формирование строки подключения
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		Config.DBUser, Config.DBPassword, Config.DBHost, Config.DBPort, Config.DBName)

	// Подключение к базе данных
	DB, err = pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v\n", err)
	}

	// Проверка подключения
	err = DB.Ping(context.Background())
	if err != nil {
		log.Fatalf("Не удалось проверить подключение к базе данных: %v\n", err)
	}

	log.Println("Успешно подключились к PostgreSQL")
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
