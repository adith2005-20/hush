package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
}

type Secret struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Project   string `json:"project"`
	Env       string `json:"environment"`
	UpdatedAt string `json:"updated_at"`
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL,
		token,
	}
}

func (c *Client) SetSecret(project, env, key, encryptedvalue string) error {
	secret := Secret{
		key,
		encryptedvalue,
		project,
		env,
		time.Now().Local().String(),
	}

	body, _ := json.Marshal(secret)
	req, err := http.NewRequest("POST", c.BaseURL+"/api/secrets", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return errors.New("Invalid response from server" + string(body))
	}

	return nil
}

func (c *Client) GetSecrets(project, env string) ([]Secret, error) {
	url := fmt.Sprintf("%s/api/secrets?project=%s&environment=%s", c.BaseURL, project, env)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("server returned a malformed response")
	}

	var secrets []Secret

	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

func (c *Client) ListProjects() ([]string, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/projects", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var projects []string

	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func (c *Client) Ping() error {
    resp, err := http.Get(c.BaseURL + "/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("server unhealthy")
    }
    return nil
}