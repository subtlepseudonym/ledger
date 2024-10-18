package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
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
	Version = "0.2.1"

	environment            string
	configPath             string
	transactionsOutputPath string
	investmentsOutputPath  string
)

func main() {
	cmd := cobra.Command{
		Use:     "plaid2csv [flags]",
		Short:   "Query plaid transaction data and output to csv",
		Version: Version,
		RunE:    run,
		SilenceUsage: true,
	}

	flags := cmd.Flags()
	flags.String("start", "", "Start date, inclusive. Format: YYYY-MM-DD")
	flags.String("end", "", "End date, inclusive. Format: YYYY-MM-DD")

	flags.String("environment", defaultEnvironment, "Environment to run in (sandbox|development|production)")
	flags.String("config", defaultConfigPath, "Config file path")
	flags.String("output-transactions", "transactions.csv", "Path for transactions output file")
	flags.String("output-investments", "investments.csv", "Path for investments output file")

	flags.Bool("clamp-semimonthly", false, "Remove transactions outside semimonthly period")
	flags.Bool("inclusive-end-date", false, "Include transactions on the end date")
	flags.Bool("sort", false, "Sort transactions by date for each account")
	flags.Bool("omit-header", false, "Omit csv header")
	flags.Bool("omit-pending", false, "Omit pending transactions")
	flags.Bool("yes", false, "Assume yes to prompts; run non-interactively")
	flags.Duration("refresh-threshold", ledger.RefreshThresholdLimit, "WARN: ($0.12/item) Request refresh if older than duration")
	flags.String("category-delimiter", ledger.DefaultCategoryDelimiter, "Delimiter for joining category hierarchy")
	flags.String("format-post-date", ledger.DefaultPostDateFormat, "Output format for transaction post date")
	flags.String("format-auth-date", ledger.DefaultAuthDateFormat, "Output format for transaction authorization date")
	flags.String("format-amount", ledger.DefaultAmountFormat, "Output format for amount")

	cmd.MarkFlagRequired("start")
	cmd.MarkFlagRequired("end")

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	yes, _ := flags.GetBool("yes")
	environment, _ := flags.GetString("environment")
	if environment == "production" && !yes {
		fmt.Println("This will run against the production environment and may incur charges. Enter 'yes' to continue")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan(); scanner.Text() != "yes" {
			return nil
		}
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

	inclusiveEndDate, _ := flags.GetBool("inclusive-end-date")
	if inclusiveEndDate {
		end = end.AddDate(0, 0, 1)
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

	transactionsOutputPath, _ := flags.GetString("output-transactions")
	transactionsOutputFile, err := os.OpenFile(transactionsOutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open transactions output file for writing: %w", err)
	}
	defer transactionsOutputFile.Close()

	investmentsOutputPath, _ := flags.GetString("output-investments")
	investmentsOutputFile, err := os.OpenFile(investmentsOutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open investments output file for writing: %w", err)
	}
	defer investmentsOutputFile.Close()

	refreshThreshold, _ := flags.GetDuration("refresh-threshold")
	activity, err := ledger.RequestActivity(config, start, end, refreshThreshold)
	if err != nil {
		return fmt.Errorf("request activity from plaid: %w", err)
	}

	omitHeader, _ := flags.GetBool("omit-header")
	transactionsOutput := csv.NewWriter(transactionsOutputFile)
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
		transactionsOutput.Write(headers)
	}

	investmentsOutput := csv.NewWriter(investmentsOutputFile)
	if !omitHeader {
		headers := []string{
			"Post Date",
			"Account",
			"Account Name",
			"Name",
			"Quantity",
			"Amount",
			"Price",
			"Transaction ID",
			"Fee",
			"Fee Currency",
			"Ticker Symbol",
			"Category",
		}
		investmentsOutput.Write(headers)
	}

	clampSemimonthly, _ := flags.GetBool("clamp-semimonthly")
	sortOutput, _ := flags.GetBool("sort")
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

	for _, item := range activity {
		itemConfig, ok := config.Items[item.ID]
		if !ok {
			log.Printf("Warning: skipping data for unknown item ID: %q\n", item.ID)
			continue
		}

		if clampSemimonthly {
			clampEndDate := end
			if end.Day() < 15 {
				clampEndDate = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, -1, end.Location())
			} else {
				clampEndDate = time.Date(end.Year(), end.Month(), 16, 0, 0, 0, -1, end.Location())
			}

			var transactions []ledger.Transaction
			for _, transaction := range item.Transactions {
				if transaction.AuthorizedDate.Time.IsZero() {
					if transaction.Date.Time.After(clampEndDate) || start.After(transaction.Date.Time) {
						continue
					}
				} else {
					if transaction.AuthorizedDate.Time.After(clampEndDate) || start.After(transaction.AuthorizedDate.Time) {
						continue
					}
				}
				transactions = append(transactions, transaction)
			}
			item.Transactions = transactions

			var investments []ledger.InvestmentTransaction
			for _, investment := range item.Investments {
				if investment.Date.Time.After(clampEndDate) || start.After(investment.Date.Time) {
					continue
				}
				investments = append(investments, investment)
			}
			item.Investments = investments
		}

		if sortOutput {
			sort.Slice(item.Transactions, func(i, j int) bool {
				return item.Transactions[j].Date.Time.After(item.Transactions[i].Date.Time)
			})
			sort.Slice(item.Investments, func(i, j int) bool {
				return item.Investments[j].Date.Time.After(item.Investments[i].Date.Time)
			})
		}

		err = ledger.WriteTransactions(itemConfig, transactionsOutput, item, options)
		if err != nil {
			return fmt.Errorf("write transactions for %q to output: %w", itemConfig.Name, err)
		}

		err = ledger.WriteInvestments(itemConfig, investmentsOutput, item, options)
		if err != nil {
			return fmt.Errorf("write investments for %q to output: %w", itemConfig.Name, err)
		}
	}

	return nil
}
