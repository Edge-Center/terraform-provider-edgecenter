package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		fmt.Println("PROJECT_ID не найден в переменных окружения")
		return
	}

	baseURL := fmt.Sprintf("https://api.edgecenter.online/cloud/v1/projects/%s", projectID)
	req, err := http.NewRequest("DELETE", baseURL, nil)
	if err != nil {
		fmt.Println("Ошибка при создании запроса:", err)
		return
	}

	apiKey := os.Getenv("EC_API_TOKEN")
	if apiKey == "" {
		fmt.Println("API ключ не найден в переменных окружения")
		return
	}

	req.Header.Set("Authorization", "APIKey "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Ошибка при выполнении запроса:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении тела ответа:", err)
		return
	}

	fmt.Printf("Статус ответа: %s\n", resp.Status)
	fmt.Printf("Тело ответа: %s\n", string(body))
}
