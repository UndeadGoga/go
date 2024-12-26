package handlers

import (
	"anonymous-chat/models"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Message представляет структуру сообщения чата
type Message struct {
	Nickname  string `json:"nickname"`
	Type      string `json:"type"`      // 'text', 'image', 'voice'
	Content   string `json:"content"`   // для текстовых сообщений
	MediaURL  string `json:"media_url"` // URL к медиафайлу
	CreatedAt string `json:"created_at"`
}

// MessageWithRoom связывает сообщение с комнатой
type MessageWithRoom struct {
	Room    string
	Message Message
}

// Client представляет клиента WebSocket
type Client struct {
	Conn *websocket.Conn
	Send chan Message // Буферизованный канал
	Room string
	Nick string
}

// Хранилище клиентов по комнатам
var clients = make(map[string]map[*Client]bool)
var broadcast = make(chan MessageWithRoom)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Инициализация мьютекса для защиты доступа к map clients
var mutex = &sync.Mutex{}

// ChatHandler обрабатывает подключение WebSocket для чата
func ChatHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	room := vars["room"]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при обновлении соединения:", err)
		return
	}

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		nickname = "Anonymous"
	}

	client := &Client{
		Conn: conn,
		Send: make(chan Message, 256), // Буферизованный канал
		Room: room,
		Nick: nickname,
	}

	// Добавление клиента в комнату
	mutex.Lock()
	if clients[room] == nil {
		clients[room] = make(map[*Client]bool)
	}
	clients[room][client] = true
	mutex.Unlock()

	log.Printf("Клиент %s подключен к комнате %s", nickname, room)

	// Отправка истории сообщений из базы данных
	sendHistory(client)

	// Запуск горутин для чтения и записи сообщений
	go client.readPump()
	go client.writePump()
}

// sendHistory отправляет историю сообщений клиенту
func sendHistory(c *Client) {
	query := `
		SELECT m.nickname, m.type, m.content, m.media_url, m.created_at 
		FROM messages m 
		JOIN rooms r ON m.room_id = r.id 
		WHERE r.name = $1 
		ORDER BY m.created_at ASC
	`
	rows, err := models.DB.Query(context.Background(), query, c.Room)
	if err != nil {
		log.Println("Ошибка при получении истории сообщений:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		var createdAt time.Time
		err := rows.Scan(&msg.Nickname, &msg.Type, &msg.Content, &msg.MediaURL, &createdAt)
		if err != nil {
			log.Println("Ошибка при сканировании строки:", err)
			continue
		}
		msg.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		select {
		case c.Send <- msg:
			// Успешная отправка сообщения
		default:
			// Если канал заполнен, пропустить отправку
			log.Printf("Канал отправки заполнен для клиента %s в комнате %s", c.Nick, c.Room)
		}
	}
}

// readPump читает сообщения от клиента и отправляет их в канал broadcast
func (c *Client) readPump() {
	defer func() {
		c.Conn.Close()
		mutex.Lock()
		delete(clients[c.Room], c)
		mutex.Unlock()
		log.Printf("Клиент %s отключен от комнаты %s", c.Nick, c.Room)
	}()

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Неожиданная ошибка закрытия: %v", err)
			}
			break
		}

		if msg.Type == "" {
			msg.Type = "text"
		}

		msg.Nickname = c.Nick
		msg.CreatedAt = getCurrentTimestamp()

		log.Printf("Получено сообщение от %s в комнате %s: %+v", c.Nick, c.Room, msg)

		broadcast <- MessageWithRoom{
			Room:    c.Room,
			Message: msg,
		}
	}
}

// writePump отправляет сообщения клиенту из канала Send
func (c *Client) writePump() {
	for msg := range c.Send {
		err := c.Conn.WriteJSON(msg)
		if err != nil {
			log.Println("Ошибка при отправке сообщения:", err)
			c.Conn.Close()
			break
		}
	}
}

// HandleMessages обрабатывает сообщения из канала broadcast
func HandleMessages() {
	for {
		msgWithRoom := <-broadcast
		room := msgWithRoom.Room
		msg := msgWithRoom.Message

		log.Printf("Обработка сообщения в комнате %s: %+v", room, msg)

		// Сохранение сообщения в базе данных
		saveMessage(room, msg)

		// Рассылка сообщения всем клиентам в комнате
		mutex.Lock()
		roomClients := clients[room]
		mutex.Unlock()

		for client := range roomClients {
			select {
			case client.Send <- msg:
				// Сообщение отправлено успешно
			default:
				// Если канал заполнен, закрыть его и удалить клиента
				close(client.Send)
				mutex.Lock()
				delete(clients[room], client)
				mutex.Unlock()
				log.Printf("Канал отправки закрыт для клиента %s в комнате %s из-за переполнения", client.Nick, room)
			}
		}
	}
}

// saveMessage сохраняет сообщение в базе данных
func saveMessage(room string, msg Message) {
	// Получение ID комнаты
	var roomID int
	err := models.DB.QueryRow(context.Background(), "SELECT id FROM rooms WHERE name=$1", room).Scan(&roomID)
	if err != nil {
		// Если комнаты нет, создать её
		err = models.DB.QueryRow(context.Background(), "INSERT INTO rooms(name) VALUES($1) RETURNING id", room).Scan(&roomID)
		if err != nil {
			log.Println("Ошибка при создании комнаты:", err)
			return
		}
	}

	// Вставка сообщения
	_, err = models.DB.Exec(context.Background(),
		"INSERT INTO messages(room_id, nickname, type, content, media_url) VALUES($1, $2, $3, $4, $5)",
		roomID, msg.Nickname, msg.Type, msg.Content, msg.MediaURL)
	if err != nil {
		log.Println("Ошибка при сохранении сообщения:", err)
	} else {
		log.Printf("Сообщение сохранено в базе данных для комнаты %s: %+v", room, msg)
	}
}

// getCurrentTimestamp возвращает текущую временную метку в формате строки
func getCurrentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// IndexHandler обрабатывает главную страницу
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		room := r.FormValue("room")
		nickname := r.FormValue("nickname")
		if room == "" || nickname == "" {
			http.Error(w, "Комната и Никнейм обязательны", http.StatusBadRequest)
			return
		}

		// Вставка комнаты в базу данных, если она еще не существует
		_, err := models.DB.Exec(context.Background(),
			"INSERT INTO rooms(name) VALUES($1) ON CONFLICT (name) DO NOTHING", room)
		if err != nil {
			log.Println("Ошибка при вставке комнаты:", err)
			http.Error(w, "Ошибка при создании комнаты", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/chat/"+room+"?nickname="+nickname, http.StatusSeeOther)
		return
	}

	// Получение списка комнат из базы данных
	rows, err := models.DB.Query(context.Background(), "SELECT name FROM rooms")
	if err != nil {
		http.Error(w, "Ошибка при получении комнат", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			rooms = append(rooms, name)
		}
	}

	// Парсинг шаблона
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Передача данных в шаблон
	data := struct {
		Rooms []string
	}{
		Rooms: rooms,
	}

	tmpl.Execute(w, data)
}

// ChatPageHandler отображает страницу чата
func ChatPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	room := vars["room"]
	nickname := r.URL.Query().Get("nickname")

	if room == "" || nickname == "" {
		http.Error(w, "Комната и Никнейм обязательны", http.StatusBadRequest)
		return
	}

	// Парсинг шаблона
	tmpl, err := template.ParseFiles("templates/chat.html")
	if err != nil {
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Передача данных в шаблон
	data := struct {
		Room     string
		Nickname string
	}{
		Room:     room,
		Nickname: nickname,
	}

	tmpl.Execute(w, data)
}

// ImageUploadHandler обрабатывает загрузку изображений
func ImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	handleFileUpload(w, r, "image", []string{"image/jpeg", "image/png"})
}

// VoiceUploadHandler обрабатывает загрузку голосовых сообщений
func VoiceUploadHandler(w http.ResponseWriter, r *http.Request) {
	handleFileUpload(w, r, "voice", []string{"audio/mpeg", "audio/wav"})
}

// handleFileUpload универсальная функция для обработки загрузки файлов
func handleFileUpload(w http.ResponseWriter, r *http.Request, fileField string, allowedTypes []string) {
	if r.Method != "POST" {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	// Ограничение размера загружаемого файла (например, 10MB)
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		log.Printf("Ошибка при разборе формы: %v", err)
		http.Error(w, "Ошибка при разборе формы", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile(fileField)
	if err != nil {
		log.Printf("Ошибка при получении файла: %v", err)
		http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверка типа файла
	fileType := handler.Header.Get("Content-Type")
	isAllowed := false
	for _, t := range allowedTypes {
		if fileType == t {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		log.Printf("Неподдерживаемый тип файла: %s", fileType)
		http.Error(w, "Неподдерживаемый тип файла", http.StatusBadRequest)
		return
	}

	// Генерация уникального имени файла
	timestamp := time.Now().UnixNano()
	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("%d%s", timestamp, ext)

	// Создание директории uploads, если не существует
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, os.ModePerm)
		if err != nil {
			log.Printf("Ошибка при создании директории для загрузок: %v", err)
			http.Error(w, "Ошибка при создании директории для загрузок", http.StatusInternalServerError)
			return
		}
	}

	// Сохранение файла
	filePath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("Ошибка при создании файла: %v", err)
		http.Error(w, "Ошибка при создании файла", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Ошибка при копировании файла: %v", err)
		http.Error(w, "Ошибка при копировании файла", http.StatusInternalServerError)
		return
	}

	// Формирование URL для доступа к файлу
	fileURL := fmt.Sprintf("/uploads/%s", filename)

	// Получение комнаты из формы
	room := r.FormValue("room")
	if room == "" {
		log.Println("Комната обязательна, но не была предоставлена")
		http.Error(w, "Комната обязательна", http.StatusBadRequest)
		return
	}

	// Определение типа сообщения
	var msgType string
	if fileField == "image" {
		msgType = "image"
	} else if fileField == "voice" {
		msgType = "voice"
	} else {
		msgType = "file"
	}

	// Создание сообщения
	msg := Message{
		Nickname:  "System",
		Type:      msgType,
		Content:   fmt.Sprintf("файл: %s", handler.Filename),
		MediaURL:  fileURL,
		CreatedAt: getCurrentTimestamp(),
	}

	log.Printf("Создание сообщения: %+v", msg)

	broadcast <- MessageWithRoom{
		Room:    room,
		Message: msg,
	}

	// Возврат URL файла и типа
	response := struct {
		MediaURL string `json:"media_url"`
		Type     string `json:"type"`
	}{
		MediaURL: fileURL,
		Type:     msgType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Файл успешно загружен: %s (%s) в комнату: %s", handler.Filename, fileType, room)
}
