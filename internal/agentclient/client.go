package agentclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"

	"github.com/s4ndyp/backai/internal/models"
	"github.com/s4ndyp/backai/internal/version"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: stringsTrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func stringsTrimRight(s, cut string) string {
	for len(s) > 0 && stringsHasSuffix(s, cut) {
		s = s[:len(s)-len(cut)]
	}
	return s
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

type RegisterResult struct {
	Agent models.Agent `json:"agent"`
	Token string       `json:"token"`
}

func (c *Client) Register(serverURL, registerKey, name, agentVersion string) (RegisterResult, error) {
	hostname := name
	body, _ := json.Marshal(map[string]string{
		"name":         name,
		"registerKey":  registerKey,
		"agentVersion": agentVersion,
		"hostname":     hostname,
		"os":           runtime.GOOS,
	})
	req, err := http.NewRequest(http.MethodPost, stringsTrimRight(serverURL, "/")+"/api/backai/agent/register", bytes.NewReader(body))
	if err != nil {
		return RegisterResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", version.AgentUserAgent())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return RegisterResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return RegisterResult{}, fmt.Errorf("register: %s", string(b))
	}
	var out RegisterResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return RegisterResult{}, err
	}
	return out, nil
}

func (c *Client) Heartbeat(agentVersion string) error {
	body, _ := json.Marshal(map[string]string{"agentVersion": agentVersion})
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/backai/agent/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat: %s", string(b))
	}
	return nil
}

func (c *Client) ClaimWork() (*models.AgentWorkItem, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/backai/agent/work/claim", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("claim: %s", string(b))
	}
	var item models.AgentWorkItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) Report(runID uuid.UUID, rep models.AgentRunReport) error {
	body, _ := json.Marshal(rep)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/backai/agent/work/"+runID.String()+"/report", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report: %s", string(b))
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Backai-Agent-Token", c.token)
	req.Header.Set("User-Agent", version.AgentUserAgent())
}
