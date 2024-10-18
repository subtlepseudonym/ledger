package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	maxTransactionCount         = 500
	plaidDomain                 = "plaid.com"
	itemGetEndpoint             = "item/get"
	transactionsEndpoint        = "transactions/get"
	transactionsRefreshEndpoint = "transactions/refresh"
	investmentsEndpoint         = "investments/transactions/get"
	investmentsRefreshEndpoint  = "investments/refresh"

	RefreshThresholdLimit = time.Hour * 168 // one week
)

type Config struct {
	Environment string                 `yaml:"environment"`
	ClientID    string                 `yaml:"client_id"`
	Secret      string                 `yaml:"secret"`
	Items       map[string]*ItemConfig `yaml:"items"` // map item ID to token and account IDs
}

type ItemConfig struct {
	Name         string            `yaml:"name"`
	Token        string            `yaml:"token"`
	Transactions map[string]string `yaml:"transactions"` // map account IDs to names
	Investments  map[string]string `yaml:"investments"`  // map account IDs to names
}

type ItemData struct {
	ID           string
	Transactions []Transaction
	Investments  []InvestmentTransaction
	Securities   map[string]Security // map security ID to security
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

func RequestActivity(config *Config, start, end time.Time, refreshThreshold time.Duration) ([]*ItemData, error) {
	items := make([]*ItemData, 0, len(config.Items))
	for itemID, itemConfig := range config.Items {
		if refreshThreshold < RefreshThresholdLimit {
			err := checkRefresh(config, itemID, itemConfig, refreshThreshold)
			if err != nil {
				return nil, fmt.Errorf("check refresh: %w", err)
			}
		}

		item := &ItemData{
			ID: itemID,
			Securities: make(map[string]Security),
		}

		if len(itemConfig.Transactions) > 0 {
			transactionsRes, err := requestItemTransactions(config, itemConfig, start, end, 0)
			if err != nil {
				return nil, fmt.Errorf("request item %q transactions: %w", itemID, err)
			}
			item.Transactions = append(item.Transactions, transactionsRes.Transactions...)

			transactionsTotal := transactionsRes.Total
			for transactionsRes.Total >= maxTransactionCount {
				transactionsRes, err = requestItemTransactions(config, itemConfig, start, end, transactionsTotal)
				if err != nil {
					return nil, fmt.Errorf("request item %q transactions: %w", itemID, err)
				}
				item.Transactions = append(item.Transactions, transactionsRes.Transactions...)
				transactionsTotal += transactionsRes.Total
			}
		}

		if len(itemConfig.Investments) > 0 {
			investmentsRes, err := requestItemInvestments(config, itemConfig, start, end, 0)
			if err != nil {
				return nil, fmt.Errorf("request item %q investments: %w", itemID, err)
			}
			item.Investments = append(item.Investments, investmentsRes.InvestmentTransactions...)
			for _, security := range investmentsRes.Securities {
				item.Securities[security.ID] = security
			}

			investmentsTotal := investmentsRes.Total
			for investmentsRes.Total >= maxTransactionCount {
				investmentsRes, err = requestItemInvestments(config, itemConfig, start, end, investmentsTotal)
				if err != nil {
					return nil, fmt.Errorf("request item %q investments: %w", itemID, err)
				}
				item.Investments = append(item.Investments, investmentsRes.InvestmentTransactions...)
				for _, security := range investmentsRes.Securities {
					item.Securities[security.ID] = security
				}
				investmentsTotal += investmentsRes.Total
			}
		}

		items = append(items, item)
	}

	return items, nil
}

func checkRefresh(config *Config, itemID string, itemConfig *ItemConfig, refreshThreshold time.Duration) error {
	now := time.Now()
	res, err := requestItem(config, itemConfig)
	if err != nil {
		return fmt.Errorf("request item: %w", err)
	}

	lastUpdate := res.Status.Investments.LastSuccessfulUpdate
	transactionsAge := now.Sub(res.Status.Transactions.LastSuccessfulUpdate)
	if !lastUpdate.IsZero() && transactionsAge >= refreshThreshold {
		log.Printf(
			"%s: item %s: last successful transactions update at %s, %s ago, requesting refresh\n",
			now.Format(time.RFC3339),
			itemID,
			res.Status.Transactions.LastSuccessfulUpdate.Format(time.RFC3339),
			transactionsAge.Round(time.Second),
		)
		_, err := requestRefresh(config, itemConfig, transactionsRefreshEndpoint)
		if err != nil {
			return fmt.Errorf("request item refresh: %w", err)
		}
	}

	lastUpdate = res.Status.Investments.LastSuccessfulUpdate
	investmentsAge := now.Sub(res.Status.Investments.LastSuccessfulUpdate)
	if !lastUpdate.IsZero() && investmentsAge >= refreshThreshold {
		log.Printf(
			"%s: item %s: last successful investments update at %s, %s ago, requesting refresh\n",
			now.Format(time.RFC3339),
			itemID,
			res.Status.Investments.LastSuccessfulUpdate.Format(time.RFC3339),
			investmentsAge.Round(time.Second),
		)
		_, err := requestRefresh(config, itemConfig, investmentsRefreshEndpoint)
		if err != nil {
			return fmt.Errorf("request item refresh: %w", err)
		}
	}

	return nil
}

func requestItem(config *Config, itemConfig *ItemConfig) (*ItemGetResponse, error) {
	request := &BasicRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: itemConfig.Token,
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", config.Environment, plaidDomain, itemGetEndpoint)
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
		log.Printf("API Error:\n%s\n", string(b))
		fallthrough
	default:
		return nil, fmt.Errorf("bad response: %s", res.Status)
	}

	var response ItemGetResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &response, nil
}

func requestRefresh(config *Config, itemConfig *ItemConfig, endpoint string) (*RefreshResponse, error) {
	request := &BasicRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: itemConfig.Token,
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", config.Environment, plaidDomain, endpoint)
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
		log.Printf("API Error:\n%s\n", string(b))
		fallthrough
	default:
		return nil, fmt.Errorf("bad response: %s", res.Status)
	}

	var response RefreshResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &response, nil
}

func requestItemTransactions(config *Config, itemConfig *ItemConfig, start, end time.Time, offset int) (*TransactionsResponse, error) {
	accounts := make([]string, 0, len(itemConfig.Transactions))
	for id := range itemConfig.Transactions {
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
		log.Printf("API Error:\n%s\n", string(b))
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

func requestItemInvestments(config *Config, itemConfig *ItemConfig, start, end time.Time, offset int) (*InvestmentTransactionsResponse, error) {
	accounts := make([]string, 0, len(itemConfig.Investments))
	for id := range itemConfig.Investments {
		accounts = append(accounts, id)
	}

	request := &InvestmentTransactionsRequest{
		ClientID:    config.ClientID,
		Secret:      config.Secret,
		AccessToken: itemConfig.Token,
		StartDate:   start.Format(time.DateOnly),
		EndDate:     end.Format(time.DateOnly),
		Options: InvestmentTransactionsRequestOptions{
			Count:       maxTransactionCount,
			Offset:      offset,
			AccountIDs:  accounts,
			AsyncUpdate: false,
		},
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/%s", config.Environment, plaidDomain, investmentsEndpoint)
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
		log.Printf("API Error:\n%s\n", string(b))
		fallthrough
	default:
		return nil, fmt.Errorf("bad response: %s", res.Status)
	}

	var response InvestmentTransactionsResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if rerr := response.Item.Error; rerr.Type != "" {
		return &response, fmt.Errorf("response error: %s %s %s", rerr.Type, rerr.Code, rerr.Message)
	}

	return &response, nil
}
