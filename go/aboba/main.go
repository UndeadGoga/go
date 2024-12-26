package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	// Создаем новый роутер Gin
	router := gin.Default()

	// Указываем маршрут для обработки GET-запросов
	router.GET("/greet", func(c *gin.Context) {
		// Получаем параметры name и age из запроса
		name := c.Query("name")
		age := c.Query("age")

		// Формируем строку ответа
		response := fmt.Sprintf("Меня зовут %s, мне %s лет", name, age)

		// Отправляем ответ
		c.String(200, response)
	})

	// Запускаем сервер на порту 8080
	router.Run(":8080")
}
