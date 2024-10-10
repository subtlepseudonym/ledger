package ledger

import (
	"encoding/json"
	"time"
)

type BasicRequest struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret"`
	AccessToken string `json:"access_token"`
}

type ItemGetResponse struct {
	Item      Item       `json:"item"`
	Status    ItemStatus `json:"status"`
	RequestID string     `json:"request_id"`
}

type RefreshResponse struct {
	RequestID string `json:"request_id"`
}

type TransactionsRequest struct {
	ClientID    string                     `json:"client_id"`
	Secret      string                     `json:"secret"`
	AccessToken string                     `json:"access_token"`
	StartDate   string                     `json:"start_date"`
	EndDate     string                     `json:"end_date"`
	Options     TransactionsRequestOptions `json:"options"`
}

type TransactionsRequestOptions struct {
	Count                      int      `json:"count"` // max 500
	Offset                     int      `json:"offset"`
	AccountIDs                 []string `json:"account_ids"`
	IncludeOriginalDescription bool     `json:"include_original_description"`
}

type TransactionsResponse struct {
	Item         Item          `json:"item"`
	Accounts     []Account     `json:"accounts"`
	Transactions []Transaction `json:"transactions"`
	RequestID    string        `json:"request_id"`
	Total        int           `json:"total_transactions"`
}

type InvestmentTransactionsRequest struct {
	ClientID    string                               `json:"client_id"`
	Secret      string                               `json:"secret"`
	AccessToken string                               `json:"access_token"`
	StartDate   string                               `json:"start_date"`
	EndDate     string                               `json:"end_date"`
	Options     InvestmentTransactionsRequestOptions `json:"options"`
}

type InvestmentTransactionsRequestOptions struct {
	Count       int      `json:"count"` // max 500
	Offset      int      `json:"offset"`
	AccountIDs  []string `json:"account_ids"`
	AsyncUpdate bool     `json:"async_update"`
}

type InvestmentTransactionsResponse struct {
	Item                      Item                    `json:"item"`
	Accounts                  []Account               `json:"accounts"`
	Securities                []Security              `json:"securities"`
	InvestmentTransactions    []InvestmentTransaction `json:"investment_transactions"`
	RequestID                 string                  `json:"request_id"`
	Total                     int                     `json:"total_investment_transactions"`
	IsInvestmentsFallbackItem bool                    `json:"is_investments_fallback_item"`
}

type Item struct {
	ID            string `json:"item_id"`
	InstitutionID string `json:"institution_id"`

	AvailableProducts []string `json:"available_products"`
	BilledProducts    []string `json:"billed_products"`
	OptionalProducts  []string `json:"optional_products"`
	Products          []string `json:"products"`

	UpdateType string   `json:"update_type"`
	Error      APIError `json:"error"`
}

type ItemStatus struct {
	Transactions struct {
		LastSuccessfulUpdate time.Time `json:"last_successful_update"`
		LastFailedUpdate     time.Time `json:"last_failed_update"`
	} `json:"transactions"`
	Investments struct {
		LastSuccessfulUpdate time.Time `json:"last_successful_update"`
		LastFailedUpdate     time.Time `json:"last_failed_update"`
	} `json:"investments"`
	LastWebhook struct {
		SentAt   time.Time `json:"sent_at"`
		CodeSent string    `json:"code_sent"`
	} `json:"last_webhook"`
}

type APIError struct {
	Type             string     `json:"error_type"`
	Code             string     `json:"error_code"`
	Message          string     `json:"error_message"`
	Display          string     `json:"display_message"`
	RequestID        string     `json:"request_id"`
	Causes           []APIError `json:"causes"`
	HTTPStatus       int        `json:"status"`
	DocumentationURL string     `json:"documentation_url"`
}

type Account struct {
	ID           string  `json:"account_id"`
	Balance      Balance `json:"balances"`
	Mask         string  `json:"mask"`
	Name         string  `json:"name"`
	OfficialName string  `json:"official_name"`
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
}

type Balance struct {
	Available          float64 `json:"available"`
	Current            float64 `json:"current"`
	Limit              float64 `json:"limit"`
	ISOCurrency        string  `json:"iso_currency_code"`
	UnofficialCurrency string  `json:"unofficial_currency_code"`
}

type Security struct {
	ID    string `json:"security_id"`
	ISIN  string `json:"isin"`
	CUSIP string `json:"cusip"`
	SEDOL string `json:"sedol"`

	InstitutionSecurityID string `json:"institution_security_id"`
	InstitutionID         string `json:"institution_id"`
	ProxySecurityID       string `json:"proxy_security_id"`

	Name             string `json:"name"`
	TickerSymbol     string `json:"ticker_symbol"`
	IsCashEquivalent bool   `json:"is_cash_equivalent"`
	Type             string `json:"type"`

	ClosePrice           float64   `json:"close_price"`
	ClosePriceAsOf       Date      `json:"close_price_as_of"`
	UpdateDatetime       time.Time `json:"update_datetime"`
	ISOCurrency          string    `json:"iso_currency_code"`
	UnofficialCurrency   string    `json:"unofficial_currency_code"`
	MarketIdentifierCode string    `json:"market_identifier_code"`
	Sector               string    `json:"sector"`
	Industry             string    `json:"industry"`
	// OptionContract OptionContract `json:"option_contract"`
}

type Holding struct {
	AccountID  string `json:"account_id"`
	SecurityID string `json:"security_id"`

	InstitutionPrice         float64   `json:"institution_price"`
	InstitutionPriceAsOf     Date      `json:"institution_price_as_of"`
	InstitutionPriceDatetime time.Time `json:"institution_price_datetime"`
	InstitutionValue         float64   `json:"institution_value"`

	CostBasis          float64 `json:"cost_basis"`
	Quantity           float64 `json:"quantity"`
	ISOCurrency        string  `json:"iso_currency_code"`
	UnofficialCurrency string  `json:"unofficial_currency_code"`
	VestedQuantity     float64 `json:"vested_quantity"`
	VestedValue        float64 `json:"vested_value"`
}

type Transaction struct {
	ID           string `json:"transaction_id"`
	Type         string `json:"transaction_type"`
	AccountID    string `json:"account_id"`
	AccountOwner string `json:"account_owner"`

	Amount             float64 `json:"amount"`
	ISOCurrency        string  `json:"iso_currency_code"`
	UnofficialCurrency string  `json:"unofficial_currency_code"`
	CheckNumber        string  `json:"check_number"`

	CategoryID string   `json:category_id"`
	Category   []string `json:"category"`

	Date           Date      `json:"date"`
	Time           time.Time `json:"datetime"`
	AuthorizedDate Date      `json:"authorized_date"`
	AuthorizedTime time.Time `json:"authorized_time"`
	Location       Location  `json:"location"`

	OriginalDescription string      `json:"original_description"`
	Name                string      `json:"name"`
	MerchantName        string      `json:"merchant_name"`
	PaymentMeta         PaymentMeta `json:"payment_meta"`
	PaymentChannel      string      `json:"payment_channel"`

	Pending              bool   `json:"pending"`
	PendingTransactionID string `json:"pending_transaction_id"`
	TransactionCode      string `json:"transaction_code"`
}

type InvestmentTransaction struct {
	ID         string `json:"investment_transaction_id"`
	AccountID  string `json:"account_id"`
	SecurityID string `json:"security_id"`

	Date     Date    `json:"date"`
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Amount   float64 `json:"amount"`
	Price    float64 `json:"price"`
	Fees     float64 `json:"fees"`
	Type     string  `json:"type"`
	Subtype  string  `json:"subtype"`

	ISOCurrency        string `json:"iso_currency_code"`
	UnofficialCurrency string `json:"unofficial_currency_code"`
}

type Date struct {
	time.Time
}

func (d Date) Format(layout string) string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(layout)
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var tmp string
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	if tmp != "" {
		date, err := time.Parse(time.DateOnly, tmp)
		if err != nil {
			return err
		}
		d.Time = date
	}
	return nil
}

type Location struct {
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	PostalCode  string  `json:"postal_code"`
	Country     string  `json:"country"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	StoreNumber string  `json:"store_number"`
}

type PaymentMeta struct {
	ReferenceNumber  string `json:"reference_number"`
	PPDID            string `json:"ppd_id"`
	Payee            string `json:"payee"`
	ByOrderOf        string `json:"by_order_of"`
	Payer            string `json:"payer"`
	PaymentMethod    string `json:"payment_method"`
	PaymentProcessor string `json:"payment_processor"`
	Reason           string `json:"reason"`
}
