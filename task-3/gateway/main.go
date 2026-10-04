package main

import (
	"fmt"
	"log"
	"net/http"
)

const serverAddress = ":8080"

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, err := fmt.Fprint(w, "pong")
	if err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

// Новый обработчик для скачивания файла
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Путь к файлу на вашем компьютере (может быть относительным или абсолютным)
	filePath := "./network_monitoring.rar" 

	// Опционально: принудительно заставляем браузер другого ПК именно скачивать файл,
	// а не пытаться открыть его в баузере (например, если это картинка или текст)
	w.Header().Set("Content-Disposition", "attachment; filename=\"network_monitoring.rar\"")

	// Отправляем файл
	http.ServeFile(w, r, filePath)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", pingHandler)
	mux.HandleFunc("/download", downloadHandler) // Регистрируем новый эндпоинт

	fmt.Printf("Gateway service started on http://localhost%s\n", serverAddress)
	fmt.Println("Для доступа с другого ПК используйте: http://<ВАШ_ЛОКАЛЬНЫЙ_IP>:8080/download")

	err := http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalf("gateway server failed: %v", err)
	}
}
