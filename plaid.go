package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	maxTransactionCount         = 500
	plaidDomain                 = "plaid.com"
	transactionsEndpoint        = "transactions/get"
	transactionsRefreshEndpoint = "transactions/refresh"
)

type Config struct {
	Environment string                 `yaml:"environment"`
	ClientID    string                 `yaml:"client_id"`
	Secret      string                 `yaml:"secret"`
	Items       map[string]*ItemConfig `yaml:"items"` // map item ID to token and account IDs
}

type ItemConfig struct {
	Name     string            `yaml:"name"`
	Token    string            `yaml:"token"`
	Accounts map[string]string `yaml:"accounts"` // map account IDs to names
}

func LoadConfig(filepath, environment string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w\n", err)
	}
	defer f.Close()

	configs := make(map[string]*Config)
	err = yaml.NewDecoder(f).Decode(configs)
	if err != nil {
		return nil, fmt.Errorf("decode config file: %w\n", err)
	}

	config, ok := configs[environment]
	if !ok {
		return nil, fmt.Errorf("unknown environment: %q", environment)
	}
	config.Environment = environment

	return config, nil
}

func RequestTransactions(config *Config, start, end time.Time, refresh bool) ([]TransactionsResponse, error) {
	responses := make([]TransactionsResponse, 0, len(config.Items))
	for _, itemConfig := range config.Items {
		if refresh {
			_, err := requestItemRefresh(config, itemConfig)
			if err != nil {
				return nil, fmt.Errorf("request item refresh: %w", err)
			}
		}

		res, err := requestItemTransactions(config, itemConfig, start, end, 0)
		if err != nil {
			return nil, fmt.Errorf("request item transactions: %w", err)
		}
		responses = append(responses, *res)

		total := res.Total
		for res.Total >= maxTransactionCount {
			res, err = requestItemTransactions(config, itemConfig, start, end, total)
			if err != nil {
				return nil, fmt.Errorf("request item transactions: %w", err)
			}
			responses = append(responses, *res)
			total += res.Total
		}
	}

	return responses, nil
}

func requestItemRefresh(config *Config, itemConfig *ItemConfig) (*TransactionsRefreshResponse, error) {
	request := &TransactionsRefreshRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: itemConfig.Token,
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", config.Environment, plaidDomain, transactionsRefreshEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read err response body: %w", err)
		}
		fmt.Printf("API Error:\n%s\n", string(b))
		fallthrough
	default:
		return nil, fmt.Errorf("bad response: %s", res.Status)
	}

	var response TransactionsRefreshResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &response, nil
}

func requestItemTransactions(config *Config, itemConfig *ItemConfig, start, end time.Time, offset int) (*TransactionsResponse, error) {
	accounts := make([]string, 0, len(itemConfig.Accounts))
	for id := range itemConfig.Accounts {
		accounts = append(accounts, id)
	}

	request := &TransactionsRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: itemConfig.Token,
		StartDate:   start.Format(time.DateOnly),
		EndDate:     end.Format(time.DateOnly),
		Options: TransactionsRequestOptions{
			Count:                      maxTransactionCount,
			Offset:                     offset,
			AccountIDs:                 accounts,
			IncludeOriginalDescription: true,
		},
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", config.Environment, plaidDomain, transactionsEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("read err response body: %w", err)
		}
		fmt.Printf("API Error:\n%s\n", string(b))
		fallthrough
	default:
		return nil, fmt.Errorf("bad response: %s", res.Status)
	}

	var response TransactionsResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if rerr := response.Item.Error; rerr.Type != "" {
		return &response, fmt.Errorf("response error: %s %s %s", rerr.Type, rerr.Code, rerr.Message)
	}

	return &response, nil
}
