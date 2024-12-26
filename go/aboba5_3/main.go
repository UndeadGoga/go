package main

import (
	"fmt"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sync"
	"time"
)

// Функция для применения фильтра размытия по Гауссу (3x3)
func applyGaussianBlur(img draw.RGBA64Image, x, y int, kernel [3][3]float64) color.RGBA64 {
	var r, g, b, a float64
	kernelSize := len(kernel)
	offset := kernelSize / 2

	// Применяем ядро свёртки к соседним пикселям
	for ky := -offset; ky <= offset; ky++ {
		for kx := -offset; kx <= offset; kx++ {
			px := x + kx
			py := y + ky
			if px >= 0 && px < img.Bounds().Max.X && py >= 0 && py < img.Bounds().Max.Y {
				originalColor := img.RGBA64At(px, py)
				weight := kernel[ky+offset][kx+offset]
				r += float64(originalColor.R) * weight
				g += float64(originalColor.G) * weight
				b += float64(originalColor.B) * weight
				a += float64(originalColor.A) * weight
			}
		}
	}

	return color.RGBA64{
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a),
	}
}

// Функция для параллельной обработки изображения с применением размытия
func filterWithGaussianBlur(img draw.RGBA64Image, wg *sync.WaitGroup, startY, endY int) {
	defer wg.Done()

	bounds := img.Bounds()
	kernel := [3][3]float64{
		{0.0625, 0.125, 0.0625},
		{0.125, 0.25, 0.125},
		{0.0625, 0.125, 0.0625},
	}

	// Применение фильтра для каждого пикселя в строках
	for y := startY; y < endY; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ { // избегаем крайние пиксели
			blurredColor := applyGaussianBlur(img, x, y, kernel)
			img.SetRGBA64(x, y, blurredColor)
		}
	}
}

func main() {
	// Открытие изображения
	file, err := os.Open("C:\\Users\\user\\Desktop\\aboba5_3\\input_image.png")
	if err != nil {
		fmt.Println("Ошибка при открытии изображения:", err)
		return
	}
	defer file.Close()

	// Декодирование изображения
	img, err := png.Decode(file)
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

	// Получаем размеры изображения
	bounds := drawImg.Bounds()
	height := bounds.Max.Y

	// Создание WaitGroup для параллельных горутин
	var wg sync.WaitGroup
	numGoroutines := 4 // Количество горутин для параллельной обработки

	// Разделение работы по строкам изображения на несколько горутин
	rowsPerGoroutine := height / numGoroutines
	for i := 0; i < numGoroutines; i++ {
		startY := i * rowsPerGoroutine
		endY := (i + 1) * rowsPerGoroutine
		if i == numGoroutines-1 { // Последняя горутина обрабатывает оставшиеся строки
			endY = height
		}

		wg.Add(1)
		go filterWithGaussianBlur(drawImg, &wg, startY, endY)
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
