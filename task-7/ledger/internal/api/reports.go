package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"ledger/internal/ledger"
)

func ReportSummaryHandler(w http.ResponseWriter, r *http.Request) {
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"from is required and must have format YYYY-MM-DD",
		)
		return
	}

	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"to is required and must have format YYYY-MM-DD",
		)
		return
	}

	if from.After(to) {
		writeError(w, http.StatusBadRequest, "from must not be after to")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	report, err := ledger.GetReportSummary(ctx, from, to)
	if err != nil {
		log.Printf("GetReportSummary: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, report)
}
