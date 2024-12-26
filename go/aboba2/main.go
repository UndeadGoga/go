package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Маршрут для сложения
	router.GET("/add", func(c *gin.Context) {
		a, b, err := parseQueryParams(c)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		result := a + b
		c.String(http.StatusOK, fmt.Sprintf("Результат сложения: %f", result))
	})

	// Маршрут для вычитания
	router.GET("/sub", func(c *gin.Context) {
		a, b, err := parseQueryParams(c)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		result := a - b
		c.String(http.StatusOK, fmt.Sprintf("Результат вычитания: %f", result))
	})

	// Маршрут для умножения
	router.GET("/mul", func(c *gin.Context) {
		a, b, err := parseQueryParams(c)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		result := a * b
		c.String(http.StatusOK, fmt.Sprintf("Результат умножения: %f", result))
	})

	// Маршрут для деления
	router.GET("/div", func(c *gin.Context) {
		a, b, err := parseQueryParams(c)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		if b == 0 {
			c.String(http.StatusBadRequest, "Ошибка: деление на ноль невозможно.")
			return
		}
		result := a / b
		c.String(http.StatusOK, fmt.Sprintf("Результат деления: %f", result))
	})

	// Запускаем сервер
	router.Run(":8080")
}

// Вспомогательная функция для извлечения параметров и их преобразования в float64
func parseQueryParams(c *gin.Context) (float64, float64, error) {
	aStr := c.Query("a")
	bStr := c.Query("b")

	// Проверка наличия параметров
	if aStr == "" || bStr == "" {
		return 0, 0, fmt.Errorf("Ошибка: оба параметра 'a' и 'b' обязательны.")
	}

	// Преобразование строк в числа
	a, err := strconv.ParseFloat(aStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Ошибка: параметр 'a' должен быть числом.")
	}

	b, err := strconv.ParseFloat(bStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("Ошибка: параметр 'b' должен быть числом.")
	}

	return a, b, nil
}
