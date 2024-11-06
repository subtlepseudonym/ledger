package ledger

import (
	"encoding/csv"
	"fmt"
	"strings"
)

const (
	DefaultOmitPending          = false
	DefaultPostDateFormat       = "2006-01-02"
	DefaultAuthDateFormat       = "2006-01-02"
	DefaultAmountFormat         = "%0.2f"
	DefaultCommodityPriceFormat = "%g"
	DefaultCategoryDelimiter    = "."
)

type WriteOptions struct {
	OmitPending          bool
	PostDateFormat       string
	AuthDateFormat       string
	AmountFormat         string
	CommodityPriceFormat string
	CategoryDelimiter    string
}

func NewWriteOptions() *WriteOptions {
	return &WriteOptions{
		OmitPending:          DefaultOmitPending,
		PostDateFormat:       DefaultPostDateFormat,
		AuthDateFormat:       DefaultAuthDateFormat,
		AmountFormat:         DefaultAmountFormat,
		CommodityPriceFormat: DefaultCommodityPriceFormat,
		CategoryDelimiter:    DefaultCategoryDelimiter,
	}
}

func WriteTransactions(itemConfig *ItemConfig, output *csv.Writer, item *ItemData, options *WriteOptions) (error, int) {
	var count int
	for _, transaction := range item.Transactions {
		if options.OmitPending && transaction.Pending {
			continue
		}

		payee := transaction.MerchantName
		if transaction.Name != "" {
			payee = transaction.Name
		}

		accountName, ok := itemConfig.Transactions[transaction.AccountID]
		if !ok {
			return fmt.Errorf("unknown account: %q", transaction.AccountID), count
		}

		currency := transaction.ISOCurrency
		if transaction.UnofficialCurrency != "" {
			currency = transaction.UnofficialCurrency
		}

		count += 1
		output.Write([]string{
			transaction.Date.Format(options.PostDateFormat),
			transaction.AuthorizedDate.Format(options.AuthDateFormat),
			accountName,
			itemConfig.Name,
			transaction.CheckNumber,
			payee,
			fmt.Sprintf(options.AmountFormat, transaction.Amount),
			currency,
			strings.Join(transaction.Category, options.CategoryDelimiter),
			transaction.ID,
		})
		if err := output.Error(); err != nil {
			return fmt.Errorf("write record: %w", err), count
		}
	}

	for _, transaction := range item.Investments {
		// FIXME: turn these into constants
		if transaction.Type != "cash" && transaction.Type != "fee" {
			continue
		}
		if transaction.Subtype == "stock distribution" {
			// the only non-currency cash subtype
			continue
		}

		security, ok := item.Securities[transaction.SecurityID]
		if !ok {
			return fmt.Errorf("unknown security: %q", transaction.SecurityID), count
		}

		accountName, ok := itemConfig.Investments[transaction.AccountID]
		if !ok {
			return fmt.Errorf("unknown account: %q", transaction.AccountID), count
		}

		currency := transaction.ISOCurrency
		if transaction.UnofficialCurrency != "" {
			currency = transaction.UnofficialCurrency
		}

		count += 1
		output.Write([]string{
			transaction.Date.Format(options.PostDateFormat),
			"",
			accountName,
			itemConfig.Name,
			"",
			security.Name,
			fmt.Sprintf(options.AmountFormat, transaction.Amount),
			currency,
			fmt.Sprintf("%s.%s", transaction.Type, transaction.Subtype),
			transaction.ID,
		})
	}

	output.Flush()
	if err := output.Error(); err != nil {
		return fmt.Errorf("flush output: %w", err), count
	}

	return nil, count
}

func WriteInvestments(itemConfig *ItemConfig, output *csv.Writer, item *ItemData, options *WriteOptions) (error, int) {
	var count int
	for _, transaction := range item.Investments {
		if transaction.Type == "cash" || transaction.Type == "fee" {
			// non-security transaction types
			continue
		}

		security, ok := item.Securities[transaction.SecurityID]
		if !ok {
			return fmt.Errorf("unknown security: %q", transaction.SecurityID), count
		}

		currency := transaction.ISOCurrency
		if transaction.UnofficialCurrency != "" {
			currency = transaction.UnofficialCurrency
		}

		accountName, ok := itemConfig.Investments[transaction.AccountID]
		if !ok {
			return fmt.Errorf("unknown account: %q", transaction.AccountID), count
		}

		category := fmt.Sprintf("%s.%s", security.Sector, security.Industry)
		if category == "." {
			category = "unknown"
		}

		count += 1
		output.Write([]string{
			transaction.Date.Format(options.PostDateFormat),
			accountName,
			itemConfig.Name,
			security.Name,
			fmt.Sprint(transaction.Quantity),
			fmt.Sprintf(options.AmountFormat, transaction.Amount),
			fmt.Sprintf(options.CommodityPriceFormat, transaction.Price),
			transaction.ID,
			fmt.Sprintf(options.AmountFormat, transaction.Fees),
			currency,
			security.TickerSymbol,
			category,
		})
		if err := output.Error(); err != nil {
			return fmt.Errorf("write record: %w", err), count
		}
	}

	output.Flush()
	if err := output.Error(); err != nil {
		return fmt.Errorf("flush output: %w", err), count
	}

	return nil, count
}
