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
	defaultEnvironment       = "sandbox" // free and (mostly) fully featured
	defaultTransactionCount  = 500       // max value
)

var (
	environment string
	configPath  string
	outputPath  string

	omitHeader        bool
	transactionCount  int
)

func main() {
	// required
	flag.StringVar(&environment, "environment", defaultEnvironment, "Environment to run in (sandbox|development|production)")
	flag.StringVar(&configPath, "config", "config.yaml", "Config file path")
	flag.StringVar(&outputPath, "output", "transactions.csv", "Path for output file")
	startDate := flag.String("start", "", "Start date, inclusive. Format: YYYY-MM-DD")
	endDate := flag.String("end", "", "End date, inclusive. Format: YYYY-MM-DD")

	// optional
	flag.BoolVar(&omitHeader, "omit-header", false, "Omit csv header")
	flag.IntVar(&transactionCount, "count", defaultTransactionCount, "Number of transactions to request, 0-500")

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

	response, err := ledger.RequestTransactions(config, start, end, transactionCount, 0)
	if err != nil {
		log.Printf("Error requesting transactions from plaid: %s\n", err)
		return
	}

	output := csv.NewWriter(outputFile)
	if !omitHeader {
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
		OmitPending: *omitPending,
		PostDateFormat: *postDateFormat,
		AuthDateFormat: *authDateFormat,
		AmountFormat: *amountFormat,
		CategoryDelimiter: *categoryDelimiter,
	}

	err = ledger.WriteTransactions(config, output, response, options)
	if err != nil {
		log.Printf("Error writing transactions to output: %s\n", err)
		return
	}

	responseTotal := response.Total
	for response.Total >= transactionCount {
		response, err = ledger.RequestTransactions(config, start, end, transactionCount, responseTotal)
		if err != nil {
			log.Printf("Error requesting transactions from plaid: %s\n", err)
			return
		}

		err = ledger.WriteTransactions(config, output, response, options)
		if err != nil {
			log.Printf("Error writing transactions to output: %s\n", err)
			return
		}
	}
}
