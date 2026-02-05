package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
)

//go:embed "Inter/Inter Variable/Inter.ttf"
var interFont []byte

//go:embed "Inter/Inter Hinted for Windows/Desktop/Inter-Bold.ttf"
var interBoldFont []byte

type Invoice struct {
	Id    string `json:"id" yaml:"id"`
	Title string `json:"title" yaml:"title"`

	Logo string `json:"logo" yaml:"logo"`
	LogoScale float64 `json:"logoScale" yaml:"logoScale"`
	From     string `json:"from" yaml:"from"`
	To       string `json:"to" yaml:"to"`
	Date     string `json:"date" yaml:"date"`
	SaleDate string `json:"saleDate" yaml:"saleDate"`
	Due      string `json:"due" yaml:"due"`
	BillingPeriod string `json:"billingPeriod" yaml:"billingPeriod"`

	Items      []string  `json:"items" yaml:"items"`
	Quantities []int     `json:"quantities" yaml:"quantities"`
	Rates      []float64 `json:"rates" yaml:"rates"`

	Tax      float64 `json:"tax" yaml:"tax"`
	TaxName  string  `json:"taxName" yaml:"taxName"`
	Discount float64 `json:"discount" yaml:"discount"`
	Paid     float64 `json:"paid" yaml:"paid"`
	Currency string  `json:"currency" yaml:"currency"`

	Lang string `json:"lang" yaml:"lang"`

	PaymentMethod string `json:"paymentMethod" yaml:"paymentMethod"`
	Bank          string `json:"bank" yaml:"bank"`
	Swift         string `json:"swift" yaml:"swift"`
	AccountNo     string `json:"accountNo" yaml:"accountNo"`

	Note string `json:"note" yaml:"note"`
}

func DefaultInvoice() Invoice {
	return Invoice{
		Id:         time.Now().Format("20060102"),
		Title:      "",
		LogoScale:  100.0,
		Rates:      []float64{25},
		Quantities: []int{2},
		Items:      []string{"Paper Cranes"},
		From:       "Project Folded, Inc.",
		To:         "Untitled Corporation, Inc.",
		// Dates use ISO format YYYY-MM-DD
		Date:       time.Now().Format("2006-01-02"),
		SaleDate:   time.Now().Format("2006-01-02"),
		Due:        time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
		BillingPeriod: "",
		Tax:      0,
		TaxName:  "",
		Discount: 0,
		Paid:     0,
		Currency: "USD",
		Lang:     "en",
	}
}

var (
	importPath     string
	file           = Invoice{}
	defaultInvoice = DefaultInvoice()
)

// LangStrings holds all translatable strings loaded from lang/<code>.json
type LangStrings struct {
	Title           string `json:"_title"`
	InvNo           string `json:"_invNo"`
	IssueDate       string `json:"_issueDate"`
	SaleDate        string `json:"_saleDate"`
	DueDate         string `json:"_dueDate"`
	BillingPeriod   string `json:"_billingPeriod"`
	Seller          string `json:"_seller"`
	Buyer           string `json:"_buyer"`
	Item            string `json:"_item"`
	Qty             string `json:"_qty"`
	UnitNet         string `json:"_unitNet"`
	TotalNet        string `json:"_totalNet"`
	Tax             string `json:"_tax"`
	NA              string `json:"_na"`
	TotalGross      string `json:"_totalGross"`
	Notes           string `json:"_notes"`
	Payment         string `json:"_payment"`
	Bank            string `json:"_bank"`
	Swift           string `json:"_swift"`
	AccountNo       string `json:"_accountNo"`
	TotalNetPrice   string `json:"_totalNetPrice"`
	Rate            string `json:"_rate"`
	Amount          string `json:"_amount"`
	Discount        string `json:"_discount"`
	TotalGrossPrice string `json:"_totalGrossPrice"`
	PaidLabel       string `json:"_paid"`
	TotalDue        string `json:"_totalDue"`
}

// langStrings is the currently loaded language pack used across the PDF generation.
var langStrings LangStrings

// englishLangValidated tracks whether we've already validated lang/en.json.
var englishLangValidated bool

// validateLang ensures that all required translation keys are present and non-empty.
func validateLang(ls *LangStrings, code string) error {
	missing := []string{}

	if ls.Title == "" {
		missing = append(missing, "_title")
	}
	if ls.InvNo == "" {
		missing = append(missing, "_invNo")
	}
	if ls.IssueDate == "" {
		missing = append(missing, "_issueDate")
	}
	if ls.SaleDate == "" {
		missing = append(missing, "_saleDate")
	}
	if ls.DueDate == "" {
		missing = append(missing, "_dueDate")
	}
	if ls.BillingPeriod == "" {
		missing = append(missing, "_billingPeriod")
	}
	if ls.Seller == "" {
		missing = append(missing, "_seller")
	}
	if ls.Buyer == "" {
		missing = append(missing, "_buyer")
	}
	if ls.Item == "" {
		missing = append(missing, "_item")
	}
	if ls.Qty == "" {
		missing = append(missing, "_qty")
	}
	if ls.UnitNet == "" {
		missing = append(missing, "_unitNet")
	}
	if ls.TotalNet == "" {
		missing = append(missing, "_totalNet")
	}
	if ls.Tax == "" {
		missing = append(missing, "_tax")
	}
	if ls.NA == "" {
		missing = append(missing, "_na")
	}
	if ls.TotalGross == "" {
		missing = append(missing, "_totalGross")
	}
	if ls.Notes == "" {
		missing = append(missing, "_notes")
	}
	if ls.Payment == "" {
		missing = append(missing, "_payment")
	}
	if ls.Bank == "" {
		missing = append(missing, "_bank")
	}
	if ls.Swift == "" {
		missing = append(missing, "_swift")
	}
	if ls.AccountNo == "" {
		missing = append(missing, "_accountNo")
	}
	if ls.TotalNetPrice == "" {
		missing = append(missing, "_totalNetPrice")
	}
	if ls.Rate == "" {
		missing = append(missing, "_rate")
	}
	if ls.Amount == "" {
		missing = append(missing, "_amount")
	}
	if ls.Discount == "" {
		missing = append(missing, "_discount")
	}
	if ls.TotalGrossPrice == "" {
		missing = append(missing, "_totalGrossPrice")
	}
	if ls.PaidLabel == "" {
		missing = append(missing, "_paid")
	}
	if ls.TotalDue == "" {
		missing = append(missing, "_totalDue")
	}

	if len(missing) > 0 {
		return fmt.Errorf("language file lang/%s.json is missing required keys: %s", code, strings.Join(missing, ", "))
	}
	return nil
}

// ensureEnglishLang ensures that lang/en.json exists and is complete. The app
// will not run without a valid English language file.
func ensureEnglishLang() error {
	if englishLangValidated {
		return nil
	}
	path := filepath.Join("lang", "en.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("required language file %s is missing: %w", path, err)
	}
	var en LangStrings
	if err := json.Unmarshal(data, &en); err != nil {
		return fmt.Errorf("unable to parse required language file %s: %w", path, err)
	}
	if err := validateLang(&en, "en"); err != nil {
		return err
	}
	englishLangValidated = true
	return nil
}

// loadLang loads the requested language and validates that all keys are present.
// English (lang/en.json) is always required and must be valid.
func loadLang(code string) error {
	if err := ensureEnglishLang(); err != nil {
		return err
	}

	if code == "" {
		code = "en"
	}

	path := filepath.Join("lang", code+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read language file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &langStrings); err != nil {
		return fmt.Errorf("unable to parse language file %s: %w", path, err)
	}
	if err := validateLang(&langStrings, code); err != nil {
		return err
	}
	return nil
}

// sanitizeFilename normalizes an invoice ID into a safe, lowercase filename.
// - lowercases everything
// - replaces spaces with '-'
// - replaces any character not in [a-z0-9._-] with '-'
func sanitizeFilename(id string) string {
	if id == "" {
		id = "invoice"
	}
	id = strings.ToLower(id)
	id = strings.ReplaceAll(id, " ", "-")
	invalid := regexp.MustCompile(`[^a-z0-9._-]`)
	id = invalid.ReplaceAllString(id, "-")
	// collapse multiple '-' into one
	multiDash := regexp.MustCompile(`-+`)
	id = multiDash.ReplaceAllString(id, "-")
	return id
}

func init() {
	generateCmd.Flags().StringVar(&importPath, "import", "", "Imported file (.json/.yaml)")
	generateCmd.Flags().StringVar(&file.Id, "id", time.Now().Format("20060102"), "ID")
	// Title defaults to empty; language file provides the visible default.
	generateCmd.Flags().StringVar(&file.Title, "title", defaultInvoice.Title, "Title")

	generateCmd.Flags().Float64SliceVarP(&file.Rates, "rate", "r", defaultInvoice.Rates, "Rates")
	generateCmd.Flags().IntSliceVarP(&file.Quantities, "quantity", "q", defaultInvoice.Quantities, "Quantities")
	generateCmd.Flags().StringSliceVarP(&file.Items, "item", "i", defaultInvoice.Items, "Items")

	generateCmd.Flags().StringVarP(&file.Logo, "logo", "l", defaultInvoice.Logo, "Company logo")
	generateCmd.Flags().StringVarP(&file.From, "from", "f", defaultInvoice.From, "Issuing company")
	generateCmd.Flags().StringVarP(&file.To, "to", "t", defaultInvoice.To, "Recipient company")
	generateCmd.Flags().StringVar(&file.Date, "date", defaultInvoice.Date, "Issue date")
	generateCmd.Flags().StringVar(&file.SaleDate, "saleDate", defaultInvoice.SaleDate, "Sale date (defaults to issue date)")
	generateCmd.Flags().StringVar(&file.Due, "due", defaultInvoice.Due, "Payment due date")
	generateCmd.Flags().StringVar(&file.BillingPeriod, "billingPeriod", defaultInvoice.BillingPeriod, "Billing period (optional, shown below due date)")

	generateCmd.Flags().Float64Var(&file.Tax, "tax", defaultInvoice.Tax, "Tax")
	generateCmd.Flags().StringVar(&file.TaxName, "taxName", defaultInvoice.TaxName, "Tax label (e.g. VAT)")
	generateCmd.Flags().Float64VarP(&file.Discount, "discount", "d", defaultInvoice.Discount, "Discount")
	generateCmd.Flags().Float64Var(&file.Paid, "paid", defaultInvoice.Paid, "Amount already paid")
	generateCmd.Flags().StringVarP(&file.Currency, "currency", "c", defaultInvoice.Currency, "Currency")
	generateCmd.Flags().StringVar(&file.Lang, "lang", defaultInvoice.Lang, "Language code (e.g. en)")

	generateCmd.Flags().StringVar(&file.PaymentMethod, "paymentMethod", "", "Method of payment")
	generateCmd.Flags().StringVar(&file.Bank, "bank", "", "Bank")
	generateCmd.Flags().StringVar(&file.Swift, "swift", "", "SWIFT")
	generateCmd.Flags().StringVar(&file.AccountNo, "accountNo", "", "Account no")

	generateCmd.Flags().StringVarP(&file.Note, "note", "n", "", "Note")
	generateCmd.Flags().Float64Var(&file.LogoScale, "logoScale", defaultInvoice.LogoScale, "Logo scale (default 100)")

	flag.Parse()
}

var rootCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Invoice generates invoices from the command line.",
	Long:  `Invoice generates invoices from the command line.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an invoice",
	Long:  `Generate an invoice`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if importPath != "" {
			err := importData(importPath, &file, cmd.Flags())
			if err != nil {
				return err
			}
		}

		// Load language strings based on requested language code
		if err := loadLang(file.Lang); err != nil {
			return err
		}

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{
			PageSize: *gopdf.PageSizeA4,
		})
		pdf.SetMargins(40, 40, 40, 40)
		pdf.AddPage()
		err := pdf.AddTTFFontData("Inter", interFont)
		if err != nil {
			return err
		}

		err = pdf.AddTTFFontData("Inter-Bold", interBoldFont)
		if err != nil {
			return err
		}

		writeLogo(&pdf, file.Logo, file.LogoScale)
		writeHeaderBlock(&pdf, file.Title, file.Id, file.Date, file.SaleDate, file.Due, file.BillingPeriod)
		writeSellerBuyerColumns(&pdf, file.From, file.To)
		writeHeaderRow(&pdf)
		writeDivider(&pdf)      // divider before items table
		subtotal := 0.0
		for i := range file.Items {
			// Determine quantity for this item; default is 1 if not provided.
			q := 1
			if len(file.Quantities) > i {
				q = file.Quantities[i]
			}
			// If quantity is explicitly set to 0, skip this item entirely.
			if q == 0 {
				continue
			}

			// Determine rate for this item; default is 0 if not provided.
			r := 0.0
			if len(file.Rates) > i {
				r = file.Rates[i]
			}

			writeRow(&pdf, file.Items[i], q, r)
			subtotal += float64(q) * r
		}
		//writeDivider(&pdf) // divider after items table
		pdf.Br(itemsToNotesGap)
		sectionY := pdf.GetY()
		if file.Note != "" || file.PaymentMethod != "" || file.Bank != "" || file.Swift != "" || file.AccountNo != "" {
			writeNotes(&pdf, file.Note, file.PaymentMethod, file.Bank, file.Swift, file.AccountNo)
		}
		writeTotals(&pdf, sectionY, subtotal, subtotal*file.Tax, subtotal*file.Discount)
		writeFooter(&pdf, file.Id)

		// Always write into ./output directory, filename based on sanitized invoice ID
		// plus the language code, e.g. 1-02-2026-en.pdf.
		outDir := "output"
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return fmt.Errorf("unable to create output directory %s: %w", outDir, err)
		}
		safeID := sanitizeFilename(file.Id)
		langCode := file.Lang
		if langCode == "" {
			langCode = "en"
		}
		langCode = strings.ToLower(langCode)
		filename := fmt.Sprintf("%s-%s.pdf", safeID, langCode)
		outputPath := filepath.Join(outDir, filename)

		err = pdf.WritePdf(outputPath)
		if err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", outputPath)

		return nil
	},
}

func main() {
	rootCmd.AddCommand(generateCmd)
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}