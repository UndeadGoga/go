package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Маршрут для обработки POST-запроса
	router.POST("/count", func(c *gin.Context) {
		// Структура для принятия JSON-запроса
		var requestBody struct {
			Text string `json:"text"`
		}

		// Проверка, что JSON корректен
		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный формат JSON"})
			return
		}

		// Подсчет символов с использованием строковых ключей
		charCount := make(map[rune]int)
		for _, char := range requestBody.Text {
			charCount[char]++
		}

		// Возврат результата в формате JSON
		c.JSON(http.StatusOK, charCount)
	})

	// Запуск сервера
	router.Run(":8080")
}
