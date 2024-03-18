package ledger

import (
	"encoding/json"
	"time"
)

type TransactionsRefreshRequest struct {
	ClientID    string `json:"client_id"`
	Secret      string `json:"secret"`
	AccessToken string `json:"access_token"`
}

type TransactionsRefreshResponse struct {
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
