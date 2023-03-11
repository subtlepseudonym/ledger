package ledger

import (
	"bytes"
	"os"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	plaidDomain          = "plaid.com"
	transactionsEndpoint = "transactions/get"
)

type Config struct {
	Environment string            `yaml:"environment"`
	ClientID    string            `yaml:"client_id"`
	Secret      string            `yaml:"secret"`
	Token       string            `yaml:"token"`
	Accounts    map[string]string `yaml:"accounts"`
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

func RequestTransactions(config *Config, start, end time.Time, count, offset int) (*TransactionsResponse, error) {
	accounts := make([]string, 0, len(config.Accounts))
	for id := range config.Accounts {
		accounts = append(accounts, id)
	}

	request := &TransactionsRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: config.Token,
		StartDate:   start.Format(time.DateOnly),
		EndDate:     end.Format(time.DateOnly),
		Options: TransactionsRequestOptions{
			Count:                      count,
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

	if res.StatusCode != http.StatusOK {
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
