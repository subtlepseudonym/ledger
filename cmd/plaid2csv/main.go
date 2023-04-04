package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/subtlepseudonym/ledger"

	"github.com/spf13/cobra"
)

const (
	defaultEnvironment = "sandbox" // free and (mostly) fully featured
	defaultConfigPath  = "~/.ledger/config.yaml"
)

var (
	Version = "0.1.0"

	environment string
	configPath  string
	outputPath  string
)

func main() {
	cmd := cobra.Command{
		Use:     "plaid2csv [flags]",
		Short:   "Query plaid transaction data and output to csv",
		Version: Version,
		RunE:    run,
	}

	flags := cmd.Flags()
	flags.String("start", "", "Start date, inclusive. Format: YYYY-MM-DD")
	flags.String("end", "", "End date, inclusive. Format: YYYY-MM-DD")

	flags.String("environment", defaultEnvironment, "Environment to run in (sandbox|development|production)")
	flags.String("config", defaultConfigPath, "Config file path")
	flags.String("output", "transactions.csv", "Path for output file")

	flags.Bool("omit-header", false, "Omit csv header")
	flags.Bool("omit-pending", false, "Omit pending transactions")
	flags.String("format-post-date", ledger.DefaultPostDateFormat, "Output format for transaction post date")
	flags.String("format-auth-date", ledger.DefaultAuthDateFormat, "Output format for transaction authorization date")
	flags.String("format-amount", ledger.DefaultAmountFormat, "Output format for amount")
	flags.String("category-delimiter", ledger.DefaultCategoryDelimiter, "Delimiter for joining category hierarchy")

	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	environment, _ := flags.GetString("environment")
	if environment == "production" {
		// TODO: prompt "you sure?"
		return fmt.Errorf("production access not yet implemented")
	}

	startDate, _ := flags.GetString("start")
	start, err := time.Parse(time.DateOnly, startDate)
	if err != nil {
		return fmt.Errorf("parse start date: %w", err)
	}

	endDate, _ := flags.GetString("end")
	end, err := time.Parse(time.DateOnly, endDate)
	if err != nil {
		return fmt.Errorf("parse end date: %w", err)
	}

	configPath, _ := flags.GetString("config")
	if configPath == defaultConfigPath {
		homePath, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get user home directory: %w", err)
		}
		configPath = strings.Replace(configPath, "~", homePath, 1)
	}

	config, err := ledger.LoadConfig(configPath, environment)
	if err != nil {
		return fmt.Errorf("load config from file: %w", err)
	}

	outputPath, _ := flags.GetString("output")
	outputFile, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open output file for writing: %w", err)
	}
	defer outputFile.Close()

	responses, err := ledger.RequestTransactions(config, start, end)
	if err != nil {
		return fmt.Errorf("request transactions from plaid: %w", err)
	}

	output := csv.NewWriter(outputFile)
	omitHeader, _ := flags.GetBool("omit-header")
	if !omitHeader {
		headers := []string{
			"Post Date",
			"Authorized Date",
			"Account",
			"Account Name",
			"Check Number",
			"Payee",
			"Amount",
			"Currency",
			"Category",
			"Transaction ID",
		}
		output.Write(headers)
	}

	omitPending, _ := flags.GetBool("omit-pending")
	postDateFormat, _ := flags.GetString("format-post-date")
	authDateFormat, _ := flags.GetString("format-auth-date")
	amountFormat, _ := flags.GetString("format-amount")
	categoryDelimiter, _ := flags.GetString("category-delimiter")

	options := &ledger.WriteOptions{
		OmitPending:       omitPending,
		PostDateFormat:    postDateFormat,
		AuthDateFormat:    authDateFormat,
		AmountFormat:      amountFormat,
		CategoryDelimiter: categoryDelimiter,
	}

	for _, response := range responses {
		itemConfig, ok := config.Items[response.Item.ID]
		if !ok {
			log.Printf("Warning: skipping response for unknown item ID: %q\n", response.Item.ID)
			continue
		}

		err = ledger.WriteTransactions(itemConfig, output, &response, options)
		if err != nil {
			return fmt.Errorf("write transactions for %q to output: %w", itemConfig.Name, err)
		}
	}

	return nil
}
