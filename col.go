// Copyright 2016 - 2019 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to
// and read from XLSX files. Support reads and writes XLSX file generated by
// Microsoft Excel™ 2007 and later. Support save file without losing original
// charts of XLSX. This library needs Go version 1.8 or later.

package excel

import "math"

// Define the default cell size and EMU unit of measurement.
const (
	defaultColWidthPixels  float64 = 64
	defaultRowHeightPixels float64 = 20
	EMU                    int     = 9525
)

// GetColVisible provides a function to get visible of a single column by given
// worksheet name and column name. For example, get visible state of column D
// in Sheet1:
//
//    visiable, err := f.GetColVisible("Sheet1", "D")
//
func (f *File) GetColVisible(sheet, col string) (bool, error) {
	visible := true
	colNum, err := ColumnNameToNumber(col)
	if err != nil {
		return visible, err
	}

	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return false, err
	}
	if xlsx.Cols == nil {
		return visible, err
	}

	for c := range xlsx.Cols.Col {
		colData := &xlsx.Cols.Col[c]
		if colData.Min <= colNum && colNum <= colData.Max {
			visible = !colData.Hidden
		}
	}
	return visible, err
}

// SetColVisible provides a function to set visible of a single column by given
// worksheet name and column name. For example, hide column D in Sheet1:
//
//    err := f.SetColVisible("Sheet1", "D", false)
//
func (f *File) SetColVisible(sheet, col string, visible bool) error {
	colNum, err := ColumnNameToNumber(col)
	if err != nil {
		return err
	}
	colData := xlsxCol{
		Min:         colNum,
		Max:         colNum,
		Hidden:      !visible,
		CustomWidth: true,
	}
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if xlsx.Cols == nil {
		cols := xlsxCols{}
		cols.Col = append(cols.Col, colData)
		xlsx.Cols = &cols
		return err
	}
	for v := range xlsx.Cols.Col {
		if xlsx.Cols.Col[v].Min <= colNum && colNum <= xlsx.Cols.Col[v].Max {
			colData = xlsx.Cols.Col[v]
		}
	}
	colData.Min = colNum
	colData.Max = colNum
	colData.Hidden = !visible
	colData.CustomWidth = true
	xlsx.Cols.Col = append(xlsx.Cols.Col, colData)
	return err
}

// GetColOutlineLevel provides a function to get outline level of a single
// column by given worksheet name and column name. For example, get outline
// level of column D in Sheet1:
//
//    level, err := f.GetColOutlineLevel("Sheet1", "D")
//
func (f *File) GetColOutlineLevel(sheet, col string) (uint8, error) {
	level := uint8(0)
	colNum, err := ColumnNameToNumber(col)
	if err != nil {
		return level, err
	}
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return 0, err
	}
	if xlsx.Cols == nil {
		return level, err
	}
	for c := range xlsx.Cols.Col {
		colData := &xlsx.Cols.Col[c]
		if colData.Min <= colNum && colNum <= colData.Max {
			level = colData.OutlineLevel
		}
	}
	return level, err
}

// SetColOutlineLevel provides a function to set outline level of a single
// column by given worksheet name and column name. For example, set outline
// level of column D in Sheet1 to 2:
//
//    err := f.SetColOutlineLevel("Sheet1", "D", 2)
//
func (f *File) SetColOutlineLevel(sheet, col string, level uint8) error {
	colNum, err := ColumnNameToNumber(col)
	if err != nil {
		return err
	}
	colData := xlsxCol{
		Min:          colNum,
		Max:          colNum,
		OutlineLevel: level,
		CustomWidth:  true,
	}
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if xlsx.Cols == nil {
		cols := xlsxCols{}
		cols.Col = append(cols.Col, colData)
		xlsx.Cols = &cols
		return err
	}
	for v := range xlsx.Cols.Col {
		if xlsx.Cols.Col[v].Min <= colNum && colNum <= xlsx.Cols.Col[v].Max {
			colData = xlsx.Cols.Col[v]
		}
	}
	colData.Min = colNum
	colData.Max = colNum
	colData.OutlineLevel = level
	colData.CustomWidth = true
	xlsx.Cols.Col = append(xlsx.Cols.Col, colData)
	return err
}

// SetColWidth provides a function to set the width of a single column or
// multiple columns. For example:
//
//    f := excelize.NewFile()
//    err := f.SetColWidth("Sheet1", "A", "H", 20)
//
func (f *File) SetColWidth(sheet, startcol, endcol string, width float64) error {
	min, err := ColumnNameToNumber(startcol)
	if err != nil {
		return err
	}
	max, err := ColumnNameToNumber(endcol)
	if err != nil {
		return err
	}
	if min > max {
		min, max = max, min
	}

	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	col := xlsxCol{
		Min:         min,
		Max:         max,
		Width:       width,
		CustomWidth: true,
	}
	if xlsx.Cols != nil {
		xlsx.Cols.Col = append(xlsx.Cols.Col, col)
	} else {
		cols := xlsxCols{}
		cols.Col = append(cols.Col, col)
		xlsx.Cols = &cols
	}
	return err
}

// positionObjectPixels calculate the vertices that define the position of a
// graphical object within the worksheet in pixels.
//
//          +------------+------------+
//          |     A      |      B     |
//    +-----+------------+------------+
//    |     |(x1,y1)     |            |
//    |  1  |(A1)._______|______      |
//    |     |    |              |     |
//    |     |    |              |     |
//    +-----+----|    OBJECT    |-----+
//    |     |    |              |     |
//    |  2  |    |______________.     |
//    |     |            |        (B2)|
//    |     |            |     (x2,y2)|
//    +-----+------------+------------+
//
// Example of an object that covers some of the area from cell A1 to B2.
//
// Based on the width and height of the object we need to calculate 8 vars:
//
//    colStart, rowStart, colEnd, rowEnd, x1, y1, x2, y2.
//
// We also calculate the absolute x and y position of the top left vertex of
// the object. This is required for images.
//
// The width and height of the cells that the object occupies can be
// variable and have to be taken into account.
//
// The values of col_start and row_start are passed in from the calling
// function. The values of col_end and row_end are calculated by
// subtracting the width and height of the object from the width and
// height of the underlying cells.
//
//    colStart        # Col containing upper left corner of object.
//    x1              # Distance to left side of object.
//
//    rowStart        # Row containing top left corner of object.
//    y1              # Distance to top of object.
//
//    colEnd          # Col containing lower right corner of object.
//    x2              # Distance to right side of object.
//
//    rowEnd          # Row containing bottom right corner of object.
//    y2              # Distance to bottom of object.
//
//    width           # Width of object frame.
//    height          # Height of object frame.
//
//    xAbs            # Absolute distance to left side of object.
//    yAbs            # Absolute distance to top side of object.
//
func (f *File) positionObjectPixels(sheet string, col, row, x1, y1, width, height int) (int, int, int, int, int, int, int, int) {
	xAbs := 0
	yAbs := 0

	// Calculate the absolute x offset of the top-left vertex.
	for colID := 1; colID <= col; colID++ {
		xAbs += f.getColWidth(sheet, colID)
	}
	xAbs += x1

	// Calculate the absolute y offset of the top-left vertex.
	// Store the column change to allow optimisations.
	for rowID := 1; rowID <= row; rowID++ {
		yAbs += f.getRowHeight(sheet, rowID)
	}
	yAbs += y1

	// Adjust start column for offsets that are greater than the col width.
	for x1 >= f.getColWidth(sheet, col) {
		x1 -= f.getColWidth(sheet, col)
		col++
	}

	// Adjust start row for offsets that are greater than the row height.
	for y1 >= f.getRowHeight(sheet, row) {
		y1 -= f.getRowHeight(sheet, row)
		row++
	}

	// Initialise end cell to the same as the start cell.
	colEnd := col
	rowEnd := row

	width += x1
	height += y1

	// Subtract the underlying cell widths to find end cell of the object.
	for width >= f.getColWidth(sheet, colEnd+1) {
		colEnd++
		width -= f.getColWidth(sheet, colEnd)
	}

	// Subtract the underlying cell heights to find end cell of the object.
	for height >= f.getRowHeight(sheet, rowEnd) {
		height -= f.getRowHeight(sheet, rowEnd)
		rowEnd++
	}

	// The end vertices are whatever is left from the width and height.
	x2 := width
	y2 := height
	return col, row, xAbs, yAbs, colEnd, rowEnd, x2, y2
}

// getColWidth provides a function to get column width in pixels by given
// sheet name and column index.
func (f *File) getColWidth(sheet string, col int) int {
	xlsx, _ := f.workSheetReader(sheet)
	if xlsx.Cols != nil {
		var width float64
		for _, v := range xlsx.Cols.Col {
			if v.Min <= col && col <= v.Max {
				width = v.Width
			}
		}
		if width != 0 {
			return int(convertColWidthToPixels(width))
		}
	}
	// Optimisation for when the column widths haven't changed.
	return int(defaultColWidthPixels)
}

// GetColWidth provides a function to get column width by given worksheet name
// and column index.
func (f *File) GetColWidth(sheet, col string) (float64, error) {
	colNum, err := ColumnNameToNumber(col)
	if err != nil {
		return defaultColWidthPixels, err
	}
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return defaultColWidthPixels, err
	}
	if xlsx.Cols != nil {
		var width float64
		for _, v := range xlsx.Cols.Col {
			if v.Min <= colNum && colNum <= v.Max {
				width = v.Width
			}
		}
		if width != 0 {
			return width, err
		}
	}
	// Optimisation for when the column widths haven't changed.
	return defaultColWidthPixels, err
}

// InsertCol provides a function to insert a new column before given column
// index. For example, create a new column before column C in Sheet1:
//
//    err := f.InsertCol("Sheet1", "C")
//
func (f *File) InsertCol(sheet, col string) error {
	num, err := ColumnNameToNumber(col)
	if err != nil {
		return err
	}
	return f.adjustHelper(sheet, columns, num, 1)
}

// RemoveCol provides a function to remove single column by given worksheet
// name and column index. For example, remove column C in Sheet1:
//
//    err := f.RemoveCol("Sheet1", "C")
//
// Use this method with caution, which will affect changes in references such
// as formulas, charts, and so on. If there is any referenced value of the
// worksheet, it will cause a file error when you open it. The excelize only
// partially updates these references currently.
func (f *File) RemoveCol(sheet, col string) error {
	num, err := ColumnNameToNumber(col)
	if err != nil {
		return err
	}

	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	for rowIdx := range xlsx.SheetData.Row {
		rowData := &xlsx.SheetData.Row[rowIdx]
		for colIdx := range rowData.C {
			colName, _, _ := SplitCellName(rowData.C[colIdx].R)
			if colName == col {
				rowData.C = append(rowData.C[:colIdx], rowData.C[colIdx+1:]...)[:len(rowData.C)-1]
				break
			}
		}
	}
	return f.adjustHelper(sheet, columns, num, -1)
}

// convertColWidthToPixels provieds function to convert the width of a cell
// from user's units to pixels. Excel rounds the column width to the nearest
// pixel. If the width hasn't been set by the user we use the default value.
// If the column is hidden it has a value of zero.
func convertColWidthToPixels(width float64) float64 {
	var padding float64 = 5
	var pixels float64
	var maxDigitWidth float64 = 7
	if width == 0 {
		return pixels
	}
	if width < 1 {
		pixels = (width * 12) + 0.5
		return math.Ceil(pixels)
	}
	pixels = (width*maxDigitWidth + 0.5) + padding
	return math.Ceil(pixels)
}
