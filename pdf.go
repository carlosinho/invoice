package main

import (
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/signintech/gopdf"
)

const (
	pageWidth            = 595.28
	quantityColumnOffset = 260
	rateColumnOffset     = 290 //unit net
	amountColumnOffset   = 360 //total net
	taxColumnOffset      = 430
	grossColumnOffset    = 480
	sellerBuyerColSplit   = 290.0
	leftColWidth          = sellerBuyerColSplit - 40 - 20   // 230
	rightColWidth         = pageWidth - 40 - sellerBuyerColSplit - 20
)

const (
	bodyFontSize   = 9
	bodyLineHeight = 15 // same as spacing between invoice date lines (issue, sale, due)
	smallGap         = 10 // small whitespace between title/number, due date/divider
	itemsToNotesGap  = 52 // gap between invoice items and notes/totals section
	subtotalLabel    = "Total net price"
	taxLabel      = "Tax"
	totalLabel    = "Total gross price"
)

func writeLogo(pdf *gopdf.GoPdf, logo string, logoScale float64) {
	if logo == "" {
		return
	}
	width, height := getImageDimension(logo)
	scaledWidth := logoScale
	scaledHeight := float64(height) * scaledWidth / float64(width)
	x := pageWidth - 40 - scaledWidth
	_ = pdf.Image(logo, x, 40, &gopdf.Rect{W: scaledWidth, H: scaledHeight})
	pdf.SetXY(40, 40)
}

func writeHeaderBlock(pdf *gopdf.GoPdf, title, id, issueDate, saleDate, dueDate, billingPeriod string) {
	if saleDate == "" {
		saleDate = issueDate
	}
	_ = pdf.SetFont("Inter-Bold", "", 24)
	pdf.SetTextColor(0, 0, 0)
	// If user provided a title in JSON/YAML/CLI, use it.
	// Otherwise, fall back to the language file value.
	headerTitle := title
	if headerTitle == "" {
		headerTitle = langStrings.Title
	}
	_ = pdf.Cell(nil, headerTitle)
	pdf.SetX(40)
	pdf.Br(38)
	pdf.SetX(40)
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(100, 100, 100)
	_ = pdf.Cell(nil, langStrings.InvNo+" ")
	_ = pdf.Cell(nil, id)
	pdf.Br(32)
	_ = pdf.Cell(nil, langStrings.IssueDate+": ")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, issueDate)
	pdf.SetTextColor(100, 100, 100)
	pdf.Br(bodyLineHeight)
	_ = pdf.Cell(nil, langStrings.SaleDate+": ")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, saleDate)
	pdf.SetTextColor(100, 100, 100)
	pdf.Br(bodyLineHeight)
	_ = pdf.Cell(nil, langStrings.DueDate+": ")
	pdf.SetTextColor(0, 0, 0)
	_ = pdf.Cell(nil, dueDate)
	if billingPeriod != "" {
		pdf.SetTextColor(100, 100, 100)
		pdf.Br(bodyLineHeight)
		_ = pdf.Cell(nil, langStrings.BillingPeriod+": ")
		pdf.SetTextColor(0, 0, 0)
		_ = pdf.Cell(nil, billingPeriod)
	}
	pdf.Br(38)
	pdf.SetStrokeColor(225, 225, 225)
	writeDivider(pdf)
	pdf.Br(36)
}

func writeSellerBuyerColumns(pdf *gopdf.GoPdf, from, to string) {
	startY := pdf.GetY()
	leftX := 40.0
	rightX := sellerBuyerColSplit

	// Left column: seller — Cell + Br(bodyLineHeight) per line so spacing matches date lines (16pt)
	pdf.SetX(leftX)
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	_ = pdf.Cell(nil, langStrings.Seller)
	pdf.Br(24)
	pdf.SetTextColor(55, 55, 55)
	formattedFrom := strings.ReplaceAll(from, `\n`, "\n")
	fromLines := strings.Split(formattedFrom, "\n")
	for i := 0; i < len(fromLines); i++ {
		pdf.SetX(leftX)
		_ = pdf.SetFont("Inter", "", bodyFontSize)
		_ = pdf.Cell(nil, fromLines[i])
		pdf.Br(bodyLineHeight)
	}
	leftBottom := pdf.GetY()

	// Right column: buyer — Cell + Br(bodyLineHeight) per line so spacing matches date lines
	// gopdf Br() resets X to left margin, so SetX(rightX) before each line
	pdf.SetXY(rightX, startY)
	pdf.SetTextColor(75, 75, 75)
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	_ = pdf.Cell(nil, langStrings.Buyer)
	pdf.Br(24)
	formattedTo := strings.ReplaceAll(to, `\n`, "\n")
	toLines := strings.Split(formattedTo, "\n")
	for i := 0; i < len(toLines); i++ {
		pdf.SetX(rightX)
		if i == 0 {
			pdf.SetTextColor(0, 0, 0)
		} else {
			pdf.SetTextColor(55, 55, 55)
		}
		_ = pdf.SetFont("Inter", "", bodyFontSize)
		_ = pdf.Cell(nil, toLines[i])
		pdf.Br(bodyLineHeight)
	}
	rightBottom := pdf.GetY()

	if leftBottom > rightBottom {
		pdf.SetY(leftBottom)
	} else {
		pdf.SetY(rightBottom)
	}
	pdf.SetX(40)
	pdf.Br(48)
}

// writeDivider draws a light horizontal divider across the content width at the current Y
func writeDivider(pdf *gopdf.GoPdf) {
	pdf.SetStrokeColor(225, 225, 225)
	pdf.Line(40, pdf.GetY(), pageWidth-40, pdf.GetY())
	pdf.Br(bodyLineHeight)
}

// writeNarrowDivider draws a shorter divider used in the totals section
func writeNarrowDivider(pdf *gopdf.GoPdf) {
	pdf.SetStrokeColor(225, 225, 225)
	y := pdf.GetY()
	pdf.Line(amountColumnOffset, y, pageWidth-40, y)
	pdf.Br(10)
}

func writeHeaderRow(pdf *gopdf.GoPdf) {
	_ = pdf.SetFont("Inter", "", bodyFontSize - 1)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, strings.ToUpper(langStrings.Item))
	pdf.SetX(quantityColumnOffset)
	_ = pdf.Cell(nil, strings.ToUpper(langStrings.Qty))
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, strings.ToUpper(langStrings.UnitNet))
	pdf.SetX(amountColumnOffset)
	_ = pdf.Cell(nil, strings.ToUpper(langStrings.TotalNet))

	baseTaxHeader := langStrings.Tax
	if file.TaxName != "" {
		baseTaxHeader = file.TaxName
	}
	pdf.SetX(taxColumnOffset)
	_ = pdf.Cell(nil, strings.ToUpper(baseTaxHeader))
	pdf.SetX(grossColumnOffset)
	_ = pdf.Cell(nil, strings.ToUpper(langStrings.TotalGross))
	pdf.Br(24)
}

func writeNotes(pdf *gopdf.GoPdf, notes, paymentMethod, bank, swift, accountNo string) {
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, langStrings.Notes)
	pdf.Br(24)
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(0, 0, 0)

	if paymentMethod != "" || bank != "" || swift != "" || accountNo != "" {
		if paymentMethod != "" {
			_ = pdf.Cell(nil, langStrings.Payment+": "+paymentMethod)
			pdf.Br(bodyLineHeight)
		}
		if bank != "" {
			_ = pdf.Cell(nil, langStrings.Bank+": "+bank)
			pdf.Br(bodyLineHeight)
		}
		if swift != "" {
			_ = pdf.Cell(nil, langStrings.Swift+": "+swift)
			pdf.Br(bodyLineHeight)
		}
		if accountNo != "" {
			_ = pdf.Cell(nil, langStrings.AccountNo+": "+accountNo)
			pdf.Br(bodyLineHeight)
		}
		if notes != "" {
			pdf.Br(bodyLineHeight)
		}
	}

	formattedNotes := strings.ReplaceAll(notes, `\n`, "\n")
	notesLines := strings.Split(formattedNotes, "\n")
	for i := 0; i < len(notesLines); i++ {
		_ = pdf.Cell(nil, notesLines[i])
		pdf.Br(bodyLineHeight)
	}

	pdf.Br(48)
}
func writeFooter(pdf *gopdf.GoPdf, id string) {
	pdf.SetY(800)

	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(55, 55, 55)
	_ = pdf.Cell(nil, id)
	pdf.SetStrokeColor(225, 225, 225)
	pdf.Line(pdf.GetX()+10, pdf.GetY()+6, 550, pdf.GetY()+6)
	pdf.Br(48)
}

func writeRow(pdf *gopdf.GoPdf, item string, quantity int, rate float64) {
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(0, 0, 0)

	// net values
	totalNet := float64(quantity) * rate
	amountNet := strconv.FormatFloat(totalNet, 'f', 2, 64)

	// wrap item name so it doesn't overlap other columns
	leftMargin := pdf.MarginLeft()
	maxItemWidth := float64(quantityColumnOffset) - 10 - leftMargin

	words := strings.Fields(item)
	var lines []string
	current := ""
	for _, w := range words {
		candidate := w
		if current != "" {
			candidate = current + " " + w
		}
		width, _ := pdf.MeasureTextWidth(candidate)
		if width <= maxItemWidth || current == "" {
			current = candidate
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	lineHeight := float64(bodyLineHeight)

	// print first line with quantities/rate/amount
	pdf.SetX(leftMargin)
	if len(lines) > 0 {
		_ = pdf.Cell(nil, lines[0])
	} else {
		_ = pdf.Cell(nil, item)
	}
	pdf.SetX(quantityColumnOffset)
	_ = pdf.Cell(nil, strconv.Itoa(quantity))
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, currencySymbols[file.Currency]+strconv.FormatFloat(rate, 'f', 2, 64))
	pdf.SetX(amountColumnOffset)
	_ = pdf.Cell(nil, currencySymbols[file.Currency]+amountNet)

	// tax rate per item (uses global tax rate) – just the value, header label is in writeHeaderRow
	pdf.SetX(taxColumnOffset)
	taxRateText := langStrings.NA
	if file.Tax != 0 {
		taxRateText = strconv.FormatFloat(file.Tax*100, 'f', 2, 64) + "%"
	}
	_ = pdf.Cell(nil, taxRateText)

	// total gross per item (net + tax amount) – header label is in writeHeaderRow
	pdf.SetX(grossColumnOffset)
	totalTaxForItem := totalNet * file.Tax
	totalGross := totalNet + totalTaxForItem
	_ = pdf.Cell(nil, currencySymbols[file.Currency]+strconv.FormatFloat(totalGross, 'f', 2, 64))

	pdf.Br(lineHeight)

	// print any wrapped continuation lines for the item name (no quantities/rates on these)
	for i := 1; i < len(lines); i++ {
		pdf.SetX(leftMargin)
		_ = pdf.Cell(nil, lines[i])
		pdf.Br(lineHeight)
	}

	// bottom padding between items so rows stay visually separated,
	// regardless of how many wrapped lines the item name used
	pdf.Br(10)
}

func writeTotals(pdf *gopdf.GoPdf, startY float64, subtotal float64, tax float64, discount float64) {
	pdf.SetY(startY)
	writeTotalWithCode(pdf, langStrings.TotalNetPrice, subtotal, false)

	// Tax lines: one for rate, one for amount
	baseTaxLabel := langStrings.Tax
	if file.TaxName != "" {
		baseTaxLabel = file.TaxName
	}
	// Tax rate (percentage or n/a)
	rateWord := langStrings.Rate
	taxRateLabel := baseTaxLabel + " " + rateWord
	naText := langStrings.NA
	taxRateValue := naText
	if file.Tax != 0 {
		taxRateValue = strconv.FormatFloat(file.Tax*100, 'f', 2, 64) + "%"
	}
	writeTotalRaw(pdf, taxRateLabel, taxRateValue)
	// Tax amount (always shown, even if 0)
	amountWord := langStrings.Amount
	taxAmountLabel := baseTaxLabel + " " + amountWord
	writeTotalWithCode(pdf, taxAmountLabel, tax, false)

	if discount > 0 {
		writeTotalWithCode(pdf, langStrings.Discount, discount, false)
	}
	// Total gross price (net + tax − discount)
	totalGross := subtotal + tax - discount
	writeTotalWithCode(pdf, langStrings.TotalGrossPrice, totalGross, false)

	// Paid (only if non-zero)
	if file.Paid != 0 {
		writeTotalWithCode(pdf, langStrings.PaidLabel, file.Paid, false)
	}

	// Total due (always shown): total gross − paid
	totalDue := totalGross - file.Paid
	writeNarrowDivider(pdf)
	writeTotalWithCode(pdf, langStrings.TotalDue, totalDue, true)
}

func writeTotal(pdf *gopdf.GoPdf, label string, total float64) {
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(amountColumnOffset + 18)
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(grossColumnOffset)
	if label == totalLabel {
		_ = pdf.SetFont("Inter-Bold", "", bodyFontSize)
	} else {
		_ = pdf.SetFont("Inter", "", bodyFontSize)
	}
	_ = pdf.Cell(nil, currencySymbols[file.Currency]+strconv.FormatFloat(total, 'f', 2, 64))
	pdf.Br(20)
}

// writeTotalWithCode formats totals with currency code (e.g. "123.45 USD") instead of symbol, used
// for the final summary lines: total net price, tax amount, total gross price.
func writeTotalWithCode(pdf *gopdf.GoPdf, label string, total float64, bold bool) {
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(amountColumnOffset + 18)
	if bold {
		_ = pdf.SetFont("Inter-Bold", "", bodyFontSize)
	} else {
		_ = pdf.SetFont("Inter", "", bodyFontSize)
	}
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(grossColumnOffset)
	value := strconv.FormatFloat(total, 'f', 2, 64) + " " + file.Currency
	_ = pdf.Cell(nil, value)
	pdf.Br(20)
}

// writeTotalRaw writes a label/value pair without currency formatting (e.g. percentages, text)
func writeTotalRaw(pdf *gopdf.GoPdf, label string, value string) {
	_ = pdf.SetFont("Inter", "", bodyFontSize)
	pdf.SetTextColor(75, 75, 75)
	pdf.SetX(amountColumnOffset + 18)
	_ = pdf.Cell(nil, label)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetX(grossColumnOffset)
	_ = pdf.Cell(nil, value)
	pdf.Br(20)
}

func getImageDimension(imagePath string) (int, int) {
	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	defer file.Close()

	image, _, err := image.DecodeConfig(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", imagePath, err)
	}
	return image.Width, image.Height
}
