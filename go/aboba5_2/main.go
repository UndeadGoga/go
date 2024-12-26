package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sync"
	"time"
)

func filterParallel(img draw.RGBA64Image, y int, wg *sync.WaitGroup) {
	defer wg.Done()
	bounds := img.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		originalColor := img.RGBA64At(x, y)
		grayValue := uint16((int(originalColor.R) + int(originalColor.G) + int(originalColor.B)) / 3)
		grayColor := color.RGBA64{R: grayValue, G: grayValue, B: grayValue, A: originalColor.A}
		img.SetRGBA64(x, y, grayColor)
	}
}

func main() {
	// Открытие изображения
	file, err := os.Open("input_image.png")
	if err != nil {
		fmt.Println("Ошибка при открытии изображения:", err)
		return
	}
	defer file.Close()

	// Декодирование изображения
	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("Ошибка при декодировании изображения:", err)
		return
	}

	// Преобразование в draw.RGBA64Image
	drawImg, ok := img.(draw.RGBA64Image)
	if !ok {
		fmt.Println("Ошибка преобразования изображения.")
		return
	}

	// Замер времени
	startTime := time.Now()

	// Создание WaitGroup для параллельных горутин
	var wg sync.WaitGroup
	bounds := drawImg.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		wg.Add(1)
		go filterParallel(drawImg, y, &wg)
	}

	// Ожидание завершения всех горутин
	wg.Wait()

	// Замер времени
	duration := time.Since(startTime)
	fmt.Println("Время обработки с использованием горутин:", duration)

	// Сохранение обработанного изображения
	outputFile, err := os.Create("output_parallel_image.png")
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		return
	}
	defer outputFile.Close()

	// Сохранение изображения в формате PNG
	err = png.Encode(outputFile, drawImg)
	if err != nil {
		fmt.Println("Ошибка при сохранении изображения:", err)
	}
}
