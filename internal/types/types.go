// Package types: internal types
package types

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/rivo/tview"
)

// AppState holds the main application state
type AppState struct {
	App        *tview.Application
	Grid       *tview.Grid
	Menu       *tview.List
	MainTable  *tview.Table
	Header     *tview.TextView
	Footer     *tview.TextView
	Client     *costexplorer.Client
	Loading    bool
	DataCache  map[string]CostData
	CacheMutex sync.RWMutex
}

// CostGroup represents a cost grouping with name and amount
type CostGroup struct {
	Name   string
	Amount float64
}

// CostData represents formatted cost data for display
type CostData struct {
	Title string
	Rows  [][]string
}
