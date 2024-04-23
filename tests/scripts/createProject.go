package main

import (
	"encoding/json"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/random"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Project struct {
	ID int `json:"id"`
}

func main() {
	baseURL := "https://api.edgecenter.online/cloud/v1/projects"
	randomProjectName := fmt.Sprintf("terraform-test-%s", random.UniqueId())
	description := fmt.Sprintf("Project for test terraform resource. Ref Name: %s", os.Getenv("GITHUB_REF_NAME"))

	payload := strings.NewReader(fmt.Sprintf(`{"name": "%s", "description": "%s"}`, randomProjectName, description))

	req, err := http.NewRequest("POST", baseURL, payload)
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

	var project Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		fmt.Println("Ошибка при разборе JSON:", err)
		return
	}

	fmt.Println("ID проекта:", project.ID)
}
