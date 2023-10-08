package main

//lint:file-ignore ST1006 This is BS
import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/samber/lo"
)

// TODO: Lot of places using indices, which can cause out-of-bounds indexing.

type BetterTable struct {
	table               table.Model
	cols                []table.Column
	rows                []table.Row
	col_sort_index      int
	searching_col_index int
	searching           string
}

func MakeBetterTable() BetterTable {
	inner_table := table.New(
		table.WithFocused(false),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	inner_table.SetStyles(s)

	better_table := BetterTable{
		table:               inner_table,
		searching_col_index: 1, // Name Column
	}

	return better_table
}

func (self *BetterTable) View() string {
	return self.table.View()
}

func (self *BetterTable) MoveUp(n int) {
	self.table.MoveUp(n)
}

func (self *BetterTable) MoveDown(n int) {
	self.table.MoveDown(n)
}

func (self *BetterTable) ColNameToIndex(col_name string) int {
	col_index := -1
	for index, col := range self.cols {
		if col_name == col.Title {
			col_index = index
			break
		}
	}

	if col_index == -1 {
		fmt.Printf("Column named '%s' is not valid!\n", col_name)
		os.Exit(-1)
	}

	return col_index
}

func (self *BetterTable) GetSelected() table.Row {
	return self.table.SelectedRow()
}

func (self *BetterTable) GetAllRowsWithValue(col_index int, value string) []table.Row {
	//TODO: unsafe slice index!
	rows := make([]table.Row, 0, 8)
	for _, row := range self.rows {
		if row[col_index] == value {
			rows = append(rows, row)
		}
	}
	return rows
}

func (self *BetterTable) SetCols(cols []table.Column) {
	self.cols = cols
	self.table.SetColumns(cols)
}

func (self *BetterTable) SetRows(rows []table.Row) {
	self.rows = rows
	self.Search(self.searching)
}

func (self *BetterTable) ClearSearch() {
	self.searching = ""
	self.Search(self.searching)
}

// Search will first filter the search, than do the ordering
// That´s why all other functions call Search.
// The searching and ordering keep alive after data changed.
func (self *BetterTable) Search(search string) {
	self.searching = search
	rows := innerSearch(self.rows, self.searching_col_index, self.searching)
	rows2, _ := innerOrderByIndex(rows, self.cols, self.col_sort_index)
	self.table.SetRows(rows2)
}

func (self *BetterTable) ResetPosition() {
	self.table.SetCursor(0)
}
func (self *BetterTable) SortByIndex(col_index int) {
	self.col_sort_index = col_index

	rows := innerSearch(self.rows, self.searching_col_index, self.searching)
	rows2, cols := innerOrderByIndex(rows, self.cols, self.col_sort_index)

	self.table.SetColumns(cols)
	self.table.SetRows(rows2)
}

func (self *BetterTable) SortBy(col_name string) {
	col_index := self.ColNameToIndex(col_name)
	self.SortByIndex(col_index)
}

func (self *BetterTable) SortByNext(index_delta int) {
	next_col_index := self.col_sort_index + index_delta
	if next_col_index < 0 {
		next_col_index = len(self.cols) - 1
	}
	if next_col_index >= len(self.cols) {
		next_col_index = 0
	}
	self.col_sort_index = next_col_index
	self.SortByIndex(next_col_index)
}

func innerSearch(rows []table.Row, searching_col_index int, search string) []table.Row {

	if len(search) == 0 {
		return rows
	}

	searchable_strings := lo.Map(rows, func(value table.Row, index int) string {
		return value[searching_col_index]
	})

	ranks := fuzzy.RankFindNormalizedFold(search, searchable_strings)

	// Not sure if this is right
	sort.SliceStable(ranks, func(i, j int) bool {
		return ranks[i].Distance < ranks[j].Distance
	})

	result := lo.Map(ranks, func(value fuzzy.Rank, index int) table.Row {
		return rows[value.OriginalIndex]
	})
	return result
}

func innerOrderByIndex(rows []table.Row, cols []table.Column, col_index int) ([]table.Row, []table.Column) {

	rows_result := make([]table.Row, len(rows))
	copy(rows_result, rows)
	sort.SliceStable(rows_result, func(i, j int) bool {
		return rows_result[i][col_index] > rows_result[j][col_index]
	})

	cols_result := make([]table.Column, len(cols))
	for index, col := range cols {
		if index == col_index {
			cols_result[index] = table.Column{
				Title: fmt.Sprintf("[%s]", col.Title),
				Width: col.Width,
			}
		} else {
			cols_result[index] = table.Column{
				Title: col.Title,
				Width: col.Width,
			}
		}
	}
	return rows_result, cols_result
}
