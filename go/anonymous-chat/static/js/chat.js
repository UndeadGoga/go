
// Получение элементов формы и полей ввода
const roomInput = document.getElementById('room'); // Элемент с id 'room'
const nicknameInput = document.getElementById('nickname'); // Элемент с id 'nickname'

const room = roomInput ? roomInput.value : ""; // Получение значения комнаты
const nickname = nicknameInput ? nicknameInput.value : "Anonymous"; // Получение значения никнейма

const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
const ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/${room}?nickname=${encodeURIComponent(nickname)}`);

const messages = document.getElementById('messages'); // Блок для отображения сообщений
const messageForm = document.getElementById('messageForm'); // Форма отправки текстовых сообщений
const messageInput = document.getElementById('messageInput'); // Поле ввода текстового сообщения

const uploadImageForm = document.getElementById('uploadImageForm'); // Форма загрузки изображений
const imageInput = document.getElementById('imageInput'); // Поле выбора изображения

const uploadVoiceForm = document.getElementById('uploadVoiceForm'); // Форма загрузки голосовых сообщений
const voiceInput = document.getElementById('voiceInput'); // Поле выбора аудиофайла

const recordButton = document.getElementById('recordButton'); // Кнопка записи аудио
const recordedAudio = document.getElementById('recordedAudio'); // Аудио-плеер для записи
const voiceForm = document.getElementById('voiceForm'); // Форма отправки записанного аудио

// Обработчики WebSocket-событий
ws.onopen = function() {
    console.log("WebSocket подключено");
};

ws.onmessage = function(event) {
    const msg = JSON.parse(event.data);
    const item = document.createElement('div');

    if (msg.type === 'text') {
        item.textContent = `[${msg.created_at}] ${msg.nickname}: ${msg.content}`;
    } else if (msg.type === 'image') {
        item.innerHTML = `[${msg.created_at}] ${msg.nickname}: <br><img src="${msg.media_url}" alt="Image" style="max-width: 300px;">`;
    } else if (msg.type === 'voice') {
        item.innerHTML = `[${msg.created_at}] ${msg.nickname}: <br><audio controls src="${msg.media_url}"></audio>`;
    } else {
        // Обработка других типов сообщений, если необходимо
        item.textContent = `[${msg.created_at}] ${msg.nickname}: ${msg.content}`;
    }

    messages.appendChild(item);
    messages.scrollTop = messages.scrollHeight;
};

ws.onerror = function(error) {
    console.error("WebSocket ошибка:", error);
};

// Отправка текстовых сообщений
messageForm.addEventListener('submit', function(e) {
    e.preventDefault();
    const text = messageInput.value.trim();
    if (text) {
        const msg = {
            type: 'text', // Указываем тип сообщения
            content: text,
        };
        ws.send(JSON.stringify(msg));
        messageInput.value = '';
    }
});

// Загрузка изображений
uploadImageForm.addEventListener('submit', function(e) {
    e.preventDefault();
    const file = imageInput.files[0];
    if (!file) {
        alert("Пожалуйста, выберите изображение для загрузки.");
        return;
    }

    const formData = new FormData();
    formData.append('image', file);
    formData.append('room', room);

    fetch('/upload-image', {
        method: 'POST',
        body: formData
    })
    .then(response => {
        if (!response.ok) {
            throw new Error("Ошибка при загрузке изображения");
        }
        return response.json();
    })
    .then(data => {
        console.log("Изображение загружено:", data);
        // Опционально, можно добавить уведомление пользователю
    })
    .catch(error => {
        console.error("Ошибка при загрузке изображения:", error);
        alert("Ошибка при загрузке изображения.");
    });

    // Сбросить выбор файла
    imageInput.value = '';
});

// Загрузка голосовых сообщений
uploadVoiceForm.addEventListener('submit', function(e) {
    e.preventDefault();
    const file = voiceInput.files[0];
    if (!file) {
        alert("Пожалуйста, выберите голосовое сообщение для загрузки.");
        return;
    }

    const formData = new FormData();
    formData.append('voice', file);
    formData.append('room', room);

    fetch('/upload-voice', {
        method: 'POST',
        body: formData
    })
    .then(response => {
        if (!response.ok) {
            throw new Error("Ошибка при загрузке голосового сообщения");
        }
        return response.json();
    })
    .then(data => {
        console.log("Голосовое сообщение загружено:", data);
        // Опционально, можно добавить уведомление пользователю
    })
    .catch(error => {
        console.error("Ошибка при загрузке голосового сообщения:", error);
        alert("Ошибка при загрузке голосового сообщения.");
    });

    // Сбросить выбор файла
    voiceInput.value = '';
});

// Запись голосовых сообщений
let mediaRecorder;
let audioChunks = [];

recordButton.addEventListener('click', function() {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
        mediaRecorder.stop();
        recordButton.textContent = "Записать голосовое сообщение";
        recordButton.classList.remove('recording');
    } else {
        navigator.mediaDevices.getUserMedia({ audio: true })
            .then(stream => {
                mediaRecorder = new MediaRecorder(stream);
                mediaRecorder.start();
                recordButton.textContent = "Остановить запись";
                recordButton.classList.add('recording');

                mediaRecorder.ondataavailable = function(e) {
                    audioChunks.push(e.data);
                }

                mediaRecorder.onstop = function() {
                    const audioBlob = new Blob(audioChunks, { type: 'audio/wav' });
                    audioChunks = [];
                    const audioUrl = URL.createObjectURL(audioBlob);
                    recordedAudio.src = audioUrl;
                    recordedAudio.style.display = 'block';
                    voiceForm.style.display = 'block';

                    // Добавление обработчика отправки голосового сообщения
                    voiceForm.addEventListener('submit', function(ev) {
                        ev.preventDefault();
                        const formData = new FormData();
                        formData.append('voice', audioBlob, 'voice.wav');
                        formData.append('room', room);

                        fetch('/upload-voice', {
                            method: 'POST',
                            body: formData
                        })
                        .then(response => {
                            if (!response.ok) {
                                throw new Error("Ошибка при загрузке голосового сообщения");
                            }
                            return response.json();
                        })
                        .then(data => {
                            console.log("Голосовое сообщение загружено:", data);
                            // Можно добавить уведомление пользователю
                        })
                        .catch(error => {
                            console.error("Ошибка при загрузке голосового сообщения:", error);
                            alert("Ошибка при загрузке голосового сообщения.");
                        });

                        // Сбросить форму и скрыть элементы
                        recordedAudio.style.display = 'none';
                        voiceForm.style.display = 'none';
                        voiceForm.reset();
                    }, { once: true }); // Обработчик добавляется только один раз
                }
            })
            .catch(err => {
                console.error("Ошибка доступа к микрофону:", err);
                alert("Не удалось получить доступ к микрофону.");
            });
    }
});
