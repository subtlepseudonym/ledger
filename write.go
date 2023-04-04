package ledger

import (
	"encoding/csv"
	"fmt"
	"strings"
)

const (
	DefaultOmitPending       = false
	DefaultPostDateFormat    = "2006-01-02"
	DefaultAuthDateFormat    = "2006-01-02"
	DefaultAmountFormat      = "%0.2f"
	DefaultCategoryDelimiter = "."
)

type WriteOptions struct {
	OmitPending       bool
	PostDateFormat    string
	AuthDateFormat    string
	AmountFormat      string
	CategoryDelimiter string
}

func NewWriteOptions() *WriteOptions {
	return &WriteOptions{
		OmitPending:       DefaultOmitPending,
		PostDateFormat:    DefaultPostDateFormat,
		AuthDateFormat:    DefaultAuthDateFormat,
		AmountFormat:      DefaultAmountFormat,
		CategoryDelimiter: DefaultCategoryDelimiter,
	}
}

func WriteTransactions(itemConfig *ItemConfig, output *csv.Writer, response *TransactionsResponse, options *WriteOptions) error {
	for _, transaction := range response.Transactions {
		if options.OmitPending && transaction.Pending {
			continue
		}

		payee := transaction.MerchantName
		if transaction.Name != "" {
			payee = transaction.Name
		}

		currency := transaction.ISOCurrency
		if transaction.UnofficialCurrency != "" {
			currency = transaction.UnofficialCurrency
		}

		accountName, ok := itemConfig.Accounts[transaction.AccountID]
		if !ok {
			return fmt.Errorf("unknown account: %q", transaction.AccountID)
		}

		output.Write([]string{
			transaction.Date.Format(options.PostDateFormat),
			transaction.AuthorizedDate.Format(options.AuthDateFormat),
			accountName,
			transaction.CheckNumber,
			payee,
			fmt.Sprintf(options.AmountFormat, transaction.Amount),
			currency,
			strings.Join(transaction.Category, options.CategoryDelimiter),
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
