package main

import (
	"log"
	"os"

	"github.com/alligatorO15/fin-tracker/internal/api"
	"github.com/alligatorO15/fin-tracker/internal/config"
	"github.com/alligatorO15/fin-tracker/internal/database"
	"github.com/alligatorO15/fin-tracker/internal/market"
	"github.com/alligatorO15/fin-tracker/internal/repository"
	"github.com/alligatorO15/fin-tracker/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// загрузка .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используются переменные окружения")
	}

	// загрузка конфигурации
	cfg := config.Load()

	// инициализация базы данных
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// запуск миграций
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Ошибка выполнения миграций: %v", err)
	}

	// инициализация репозиториев
	repos := repository.NewRepositories(db)

	// инициализация провайдера рыночных данных
	marketProvider := market.NewMultiProvider(cfg)

	// инициализация сервисов
	services := service.NewServices(repos, marketProvider, cfg)

	// инициализация и запуск API сервера
	server := api.NewServer(cfg, services)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Запуск сервера FinTracker на порту %s", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
