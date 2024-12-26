package main

import (
	"log"
	"net/http"

	"anonymous-chat/handlers"
	"anonymous-chat/models"

	"github.com/gorilla/mux"
)

func main() {
	// Инициализация базы данных
	models.InitDB()
	defer models.DB.Close()

	// Создание нового маршрутизатора
	router := mux.NewRouter()

	// Маршруты для WebSocket и страниц
	router.HandleFunc("/ws/{room}", handlers.ChatHandler)
	router.HandleFunc("/chat/{room}", handlers.ChatPageHandler)
	router.HandleFunc("/", handlers.IndexHandler).Methods("GET", "POST")

	// Маршруты для загрузки файлов
	router.HandleFunc("/upload-image", handlers.ImageUploadHandler).Methods("POST")
	router.HandleFunc("/upload-voice", handlers.VoiceUploadHandler).Methods("POST")

	// Обслуживание статических файлов
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Обслуживание загруженных файлов
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads/"))))

	// Запуск обработчика сообщений
	go handlers.HandleMessages()

	// Запуск сервера
	log.Println("Сервер запущен на порту:", models.Config.Port)
	err := http.ListenAndServe(":"+models.Config.Port, router)
	if err != nil {
		log.Fatal("Ошибка запуска сервера: ", err)
	}
}
