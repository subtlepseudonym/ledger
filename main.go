package ledger

import "time"

// LineItem is any item that has monetary value. Additional getters
// are included so that it can be identified and described
type LineItem interface {
	ID() string // unique
	Name() string
	Description() string // FIXME: optional? that's more of a db concern
	Value() int // in cents
}

// Expense is a basic LineItem that includes the date when the
// expense occurred
type Expense struct {
	id          string
	name        string
	description string
	value       int

	Date time.Time // date of charge
}

// BudgetedExpense is an Expense that draws funds from a budget
// rather than income
// FIXME: the concepts of `income` and `budget` are not yet rigorously defined
type BudgetedExpense struct {
	Expense
	Budget *Budget // budget from which expense draws funds
}

// Category is a nestable LineItem that contains other LineItem structs
type Category interface {
	LineItem
	Items() []LineItem
	AddItem(LineItem)
}

// Budget is a Category that can provide funds for BudgetedExpense structs
type Budget struct {
	id          string
	name        string
	description string
	total       int
	items       []LineItem
}

func (b *Budget) AddItem(item LineItem) {
	b.items = append(b.items, item)
	b.total += item.Value()
}

func (b *Budget) Items() []LineItem {
	return b.items
}

// Ledger should have specific fields for top-level concerns
// general health of finances, value remaining in budgets,
// total cost by category, etc
type Ledger struct {
	Income   Category
	// FIXME: may want Expenses and Budgets to be map[string]Object
	Expenses []Category
	Budgets  []Budget
}
