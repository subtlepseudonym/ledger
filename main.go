package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string            `yaml:"environment"`
	ClientID    string            `yaml:"client_id"`
	Secret      string            `yaml:"secret"`
	Token       string            `yaml:"token"`
	Accounts    map[string]string `yaml:"accounts"`
}

const (
	defaultTransactionCount  = 500 // max value
	defaultCategoryDelimiter = "."

	defaultDateFormat   = "2006-01-02"
	defaultAmountFormat = "%0.2f"

	plaidDomain          = "plaid.com"
	transactionsEndpoint = "transactions/get"
)

var (
	configPath string
	outputPath string

	ignorePending     bool
	transactionCount  int
	categoryDelimiter string

	postDateFormat string
	authDateFormat string
	amountFormat   string
)

func main() {
	// required
	flag.StringVar(&configPath, "config", "config.yaml", "Config file path")
	flag.StringVar(&outputPath, "output", "transactions.csv", "Path for output file")
	startDate := flag.String("start", "", "Start date, inclusive. Format: YYYY-MM-DD")
	endDate := flag.String("end", "", "End date, inclusive. Format: YYYY-MM-DD")

	// optional
	flag.BoolVar(&ignorePending, "ignore-pending", false, "Omit pending transactions")
	flag.IntVar(&transactionCount, "count", defaultTransactionCount, "Number of transactions to request, 0-500")
	flag.StringVar(&categoryDelimiter, "category-delimiter", defaultCategoryDelimiter, "Delimiter for joining category hierarchy")

	flag.StringVar(&postDateFormat, "format-post-date", defaultDateFormat, "Output format for transaction post date")
	flag.StringVar(&authDateFormat, "format-auth-date", defaultDateFormat, "Output format for transaction authorization date")
	flag.StringVar(&amountFormat, "format-amount", defaultAmountFormat, "Output format for amount")
	flag.Parse()

	start, err := time.Parse(time.DateOnly, *startDate)
	if err != nil {
		log.Printf("Error parsing start date: %s\n", err)
		return
	}

	end, err := time.Parse(time.DateOnly, *endDate)
	if err != nil {
		log.Printf("Error parsing end date: %s\n", err)
		return
	}

	f, err := os.Open(configPath)
	if err != nil {
		log.Printf("Error opening config file: %s\n", err)
		return
	}
	defer f.Close()

	var config Config
	err = yaml.NewDecoder(f).Decode(&config)
	if err != nil {
		log.Printf("Error decoding config file: %s\n", err)
		return
	}

	outputFile, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening output file for writing: %s\n", err)
		return
	}
	defer outputFile.Close()

	response, err := RequestTransactions(&config, start, end, transactionCount, 0)
	if err != nil {
		log.Printf("Error requesting transactions from plaid: %s\n", err)
		return
	}

	output := csv.NewWriter(outputFile)
	headers := []string{
		"Post Date",
		"Authorized Date",
		"Account",
		"Check Number",
		"Payee",
		"Amount",
		"Currency",
		"Category",
		"Transaction ID",
	}
	output.Write(headers)
	err = WriteTransactions(output, response, &config)
	if err != nil {
		log.Printf("Error writing transactions to output: %s\n", err)
		return
	}

	responseTotal := response.Total
	for response.Total >= transactionCount {
		response, err = RequestTransactions(&config, start, end, transactionCount, responseTotal)
		if err != nil {
			log.Printf("Error requesting transactions from plaid: %s\n", err)
			return
		}

		err = WriteTransactions(output, response, &config)
		if err != nil {
			log.Printf("Error writing transactions to output: %s\n", err)
			return
		}
	}
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

func WriteTransactions(output *csv.Writer, response *TransactionsResponse, config *Config) error {
	for _, transaction := range response.Transactions {
		if ignorePending && transaction.Pending {
			continue
		}

		payee := transaction.Name
		if transaction.MerchantName != "" {
			payee = transaction.MerchantName
		}

		currency := transaction.ISOCurrency
		if transaction.UnofficialCurrency != "" {
			currency = transaction.UnofficialCurrency
		}

		output.Write([]string{
			transaction.Date.Format(postDateFormat),
			transaction.AuthorizedDate.Format(authDateFormat),
			config.Accounts[transaction.AccountID],
			transaction.CheckNumber,
			payee,
			fmt.Sprintf(amountFormat, transaction.Amount),
			currency,
			strings.Join(transaction.Category, categoryDelimiter),
			transaction.ID,
		})
		if err := output.Error(); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	output.Flush()
	if err := output.Error(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}
