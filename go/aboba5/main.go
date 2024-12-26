package main

import "fmt"

func count(c chan int) {
	for num := range c {
		fmt.Println("Получено число:", num)
	}
}

func main() {
	c := make(chan int)

	// Запуск горутины
	go count(c)

	// Отправка чисел в канал
	for i := 1; i <= 5; i++ {
		c <- i
	}

	// Закрытие канала
	close(c)
}
