package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// InterestRate represents a bank interest rate entry
type InterestRate struct {
	ID            int64           `db:"id" json:"id"`
	BankCode      string          `db:"bank_code" json:"bankCode"`
	BankName      string          `db:"bank_name" json:"bankName"`
	BankLogo      string          `db:"bank_logo" json:"bankLogo,omitempty"`
	ProductType   string          `db:"product_type" json:"productType"` // deposit, loan, mortgage
	TermMonths    int             `db:"term_months" json:"termMonths"`
	TermLabel     string          `db:"term_label" json:"termLabel"` // "1 tháng", "3 tháng", etc.
	Rate          decimal.Decimal `db:"rate" json:"rate"`
	MinAmount     decimal.Decimal `db:"min_amount" json:"minAmount,omitempty"`
	MaxAmount     decimal.Decimal `db:"max_amount" json:"maxAmount,omitempty"`
	Currency      string          `db:"currency" json:"currency"`
	EffectiveDate time.Time       `db:"effective_date" json:"effectiveDate"`
	ScrapedAt     time.Time       `db:"scraped_at" json:"scrapedAt"`
	CreatedAt     time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updatedAt"`
}

// Bank represents a Vietnamese bank
type Bank struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	NameVi  string `json:"nameVi"`
	Logo    string `json:"logo"`
	Website string `json:"website"`
}

// VietnameseBanks is a list of major Vietnamese banks
// Logo paths are relative to frontend public folder
var VietnameseBanks = []Bank{
	{Code: "vcb", Name: "Vietcombank", NameVi: "Ngân hàng TMCP Ngoại thương Việt Nam", Logo: "/logos/vcb.svg", Website: "https://www.vietcombank.com.vn"},
	{Code: "tcb", Name: "Techcombank", NameVi: "Ngân hàng TMCP Kỹ thương Việt Nam", Logo: "/logos/tcb.svg", Website: "https://techcombank.com"},
	{Code: "mb", Name: "MB Bank", NameVi: "Ngân hàng TMCP Quân đội", Logo: "/logos/mb.svg", Website: "https://www.mbbank.com.vn"},
	{Code: "bidv", Name: "BIDV", NameVi: "Ngân hàng TMCP Đầu tư và Phát triển Việt Nam", Logo: "/logos/bidv.svg", Website: "https://www.bidv.com.vn"},
	{Code: "agribank", Name: "Agribank", NameVi: "Ngân hàng Nông nghiệp và Phát triển Nông thôn", Logo: "/logos/agribank.svg", Website: "https://www.agribank.com.vn"},
	{Code: "vpbank", Name: "VPBank", NameVi: "Ngân hàng TMCP Việt Nam Thịnh Vượng", Logo: "/logos/vpbank.svg", Website: "https://www.vpbank.com.vn"},
	{Code: "acb", Name: "ACB", NameVi: "Ngân hàng TMCP Á Châu", Logo: "/logos/acb.svg", Website: "https://www.acb.com.vn"},
	{Code: "sacombank", Name: "Sacombank", NameVi: "Ngân hàng TMCP Sài Gòn Thương Tín", Logo: "/logos/sacombank.svg", Website: "https://www.sacombank.com.vn"},
	{Code: "tpbank", Name: "TPBank", NameVi: "Ngân hàng TMCP Tiên Phong", Logo: "/logos/tpbank.svg", Website: "https://tpb.vn"},
	{Code: "hdbank", Name: "HDBank", NameVi: "Ngân hàng TMCP Phát triển TP.HCM", Logo: "/logos/hdbank.svg", Website: "https://www.hdbank.com.vn"},
}

// StandardTerms defines common deposit term periods in months
var StandardTerms = []struct {
	Months int
	Label  string
}{
	{0, "Không kỳ hạn"},
	{1, "1 tháng"},
	{3, "3 tháng"},
	{6, "6 tháng"},
	{9, "9 tháng"},
	{12, "12 tháng"},
	{18, "18 tháng"},
	{24, "24 tháng"},
	{36, "36 tháng"},
}
