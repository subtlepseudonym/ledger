package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/subtlepseudonym/ledger"
)

const (
	defaultEnvironment = "sandbox" // free and (mostly) fully featured
)

var (
	environment string
	configPath  string
	outputPath  string
)

func main() {
	// required
	flag.StringVar(&environment, "environment", defaultEnvironment, "Environment to run in (sandbox|development|production)")
	flag.StringVar(&configPath, "config", "config.yaml", "Config file path")
	flag.StringVar(&outputPath, "output", "transactions.csv", "Path for output file")
	startDate := flag.String("start", "", "Start date, inclusive. Format: YYYY-MM-DD")
	endDate := flag.String("end", "", "End date, inclusive. Format: YYYY-MM-DD")

	// optional
	omitHeader := flag.Bool("omit-header", false, "Omit csv header")
	omitPending := flag.Bool("omit-pending", false, "Omit pending transactions")
	postDateFormat := flag.String("format-post-date", ledger.DefaultPostDateFormat, "Output format for transaction post date")
	authDateFormat := flag.String("format-auth-date", ledger.DefaultAuthDateFormat, "Output format for transaction authorization date")
	amountFormat := flag.String("format-amount", ledger.DefaultAmountFormat, "Output format for amount")
	categoryDelimiter := flag.String("category-delimiter", ledger.DefaultCategoryDelimiter, "Delimiter for joining category hierarchy")
	flag.Parse()

	if environment == "production" {
		// TODO: prompt "you sure?"
		fmt.Println("Error: production access not yet implemented")
		return
	}

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

	config, err := ledger.LoadConfig(configPath, environment)
	if err != nil {
		log.Printf("Error loading config from file: %s\n", err)
		return
	}

	outputFile, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening output file for writing: %s\n", err)
		return
	}
	defer outputFile.Close()

	responses, err := ledger.RequestTransactions(config, start, end)
	if err != nil {
		log.Printf("Error requesting transactions from plaid: %s\n", err)
		return
	}

	output := csv.NewWriter(outputFile)
	if !*omitHeader {
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
	}

	options := &ledger.WriteOptions{
		OmitPending:       *omitPending,
		PostDateFormat:    *postDateFormat,
		AuthDateFormat:    *authDateFormat,
		AmountFormat:      *amountFormat,
		CategoryDelimiter: *categoryDelimiter,
	}

	for _, response := range responses {
		itemConfig, ok := config.Items[response.Item.ID]
		if !ok {
			log.Printf("Warning: skipping response for unknown item ID: %q\n", response.Item.ID)
			continue
		}

		err = ledger.WriteTransactions(itemConfig, output, &response, options)
		if err != nil {
			log.Printf("Error writing transactions for %q to output: %s\n", itemConfig.Name, err)
			return
		}
	}
}
