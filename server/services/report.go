package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

// SettlementReport renders the PDF for a FULLY-SETTLED group (req #19): trip
// details, full expense list, per-member balances, and the payment plan with
// each payment's final status and who confirmed/finalized it (from the audit
// trail).
func (s *Services) SettlementReport(ctx context.Context, id types.Identity, groupID string) ([]byte, types.APIError) {
	summary, apiErr := s.GroupSummary(ctx, id, groupID)
	if apiErr != nil {
		return nil, apiErr
	}
	if summary.Status != types.GroupSettled {
		return nil, types.NewConflictError("the report is available once the group is fully settled")
	}

	expenses, err := s.q.ListExpenses(ctx, repo.ListExpensesParams{GroupID: groupID})
	if err != nil {
		s.logger.Errorw("report: list expenses", "error", err)
		return nil, types.NewServerError()
	}
	payments, err := s.q.ListPayments(ctx, groupID)
	if err != nil {
		s.logger.Errorw("report: list payments", "error", err)
		return nil, types.NewServerError()
	}
	actors, err := s.q.ListPaymentAuditActors(ctx, groupID)
	if err != nil {
		s.logger.Errorw("report: list payment actors", "error", err)
		return nil, types.NewServerError()
	}

	emails := map[string]string{}
	for _, m := range summary.Members {
		emails[m.UserID] = m.Email
	}
	lookup := func(userID string) string {
		if e, ok := emails[userID]; ok {
			return e
		}
		p, err := s.principalByUserID(ctx, userID)
		if err != nil {
			return userID
		}
		emails[userID] = p.Email
		return p.Email
	}

	confirmedBy := map[string]string{}
	finalizedBy := map[string]string{}
	for _, a := range actors {
		if a.ActorUserID == nil {
			continue
		}
		switch {
		case a.Action == "payment.creditor_confirmed":
			confirmedBy[a.PaymentID] = lookup(*a.ActorUserID)
		case strings.HasPrefix(a.Action, "payment.settled"):
			by := lookup(*a.ActorUserID)
			if strings.HasSuffix(a.Action, ".override") {
				by += " (override)"
			}
			finalizedBy[a.PaymentID] = by
		}
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Settlement Report — "+summary.Name, true)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, "Settlement Report")
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Trip: %s", summary.Name))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Dates: %s to %s", summary.StartDate.Format("2006-01-02"), summary.EndDate.Format("2006-01-02")))
	pdf.Ln(6)
	pdf.Cell(0, 6, fmt.Sprintf("Status: fully settled  |  Members: %d  |  Total spent: %s", summary.MemberCount, omr(summary.TotalSpent)))
	pdf.Ln(10)

	section := func(title string) {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.Cell(0, 8, title)
		pdf.Ln(9)
	}
	header := func(widths []float64, cols []string) {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(230, 230, 230)
		for i, c := range cols {
			pdf.CellFormat(widths[i], 7, c, "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Helvetica", "", 9)
	}
	row := func(widths []float64, cols []string) {
		for i, c := range cols {
			pdf.CellFormat(widths[i], 6, c, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	section("Per-member balances")
	w := []float64{70, 40, 40, 40}
	header(w, []string{"Member", "Paid", "Fair share", "Net"})
	for _, m := range summary.Members {
		row(w, []string{clip(m.Email, 40), omr(m.Paid), omr(m.FairShare), omr(m.Net)})
	}
	pdf.Ln(6)

	section("Expenses")
	w = []float64{26, 48, 22, 28, 66}
	header(w, []string{"Date", "Paid by", "Category", "Amount", "Description"})
	for _, e := range expenses {
		row(w, []string{e.OccurredOn, clip(lookup(e.PaidBy), 27), string(e.Category), omr(e.AmountBaisa), clip(truncateDescription(e.Description), 38)})
	}
	if len(expenses) == 0 {
		row(w, []string{"-", "-", "-", "-", "no expenses"})
	}
	pdf.Ln(6)

	section("Payment plan")
	w = []float64{42, 42, 26, 20, 30, 30}
	header(w, []string{"From", "To", "Amount", "Status", "Confirmed by", "Finalized by"})
	for _, p := range payments {
		row(w, []string{
			clip(lookup(p.FromUserID), 23), clip(lookup(p.ToUserID), 23), omr(p.AmountBaisa), string(p.Status),
			clip(orDash(confirmedBy[p.ID]), 16), clip(orDash(finalizedBy[p.ID]), 16),
		})
	}
	if len(payments) == 0 {
		row(w, []string{"-", "-", "-", "-", "-", "no transfers were needed"})
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		s.logger.Errorw("report: render pdf", "error", err)
		return nil, types.NewServerError()
	}
	return buf.Bytes(), nil
}

// omr renders integer baisa as OMR without any float math.
func omr(baisa int64) string {
	sign := ""
	if baisa < 0 {
		sign = "-"
		baisa = -baisa
	}
	return fmt.Sprintf("%s%d.%03d OMR", sign, baisa/1000, baisa%1000)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// clip keeps a cell's text inside its column width.
func clip(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-2]) + ".."
}
