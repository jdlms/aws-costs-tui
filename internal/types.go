package internal

import (
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/rivo/tview"
)

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

type CostGroup struct {
	Name   string
	Amount float64
}

type CostData struct {
	Title string
	Rows  [][]string
}
