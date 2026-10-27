package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChecklistResult struct {
	ID          int64   `json:"id"`
	RuleID      int64   `json:"rule_id"`
	RuleKey     string  `json:"rule_key"`
	Description string  `json:"description"`
	FieldKey    string  `json:"field_key"`
	Status      string  `json:"status"`
	Reason      string  `json:"reason"`
	Evidence    *string `json:"evidence"`
}

type ChecklistSummary struct {
	Total       int `json:"total"`
	Fulfilled   int `json:"fulfilled"`
	Missing     int `json:"missing"`
	NeedsReview int `json:"needs_review"`
}

type ChecklistEvaluation struct {
	PARequestID int64             `json:"pa_request_id"`
	Results     []ChecklistResult `json:"results"`
	Summary     ChecklistSummary  `json:"summary"`
}

type checklistRule struct {
	ID          int64
	RuleKey     string
	Description string
	FieldKey    string
}

type extractedField struct {
	Value              *string
	VerificationStatus string
	EvidenceText       *string
}

// EvaluateChecklist evaluates required fields against the checklist
// version assigned to a PA request, then persists the results.
func EvaluateChecklist(
	ctx context.Context,
	pool *pgxpool.Pool,
	requestID int64,
	reviewerIDs ...int64,
) (*ChecklistEvaluation, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin checklist evaluation: %w", err)
	}
	defer tx.Rollback(ctx)

	evaluation, err := EvaluateChecklistInTx(ctx, tx, requestID, reviewerIDs...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit checklist evaluation: %w", err)
	}
	return evaluation, nil
}

func EvaluateChecklistInTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID int64,
	reviewerIDs ...int64,
) (*ChecklistEvaluation, error) {
	var reviewerID *int64
	if len(reviewerIDs) > 0 && reviewerIDs[0] > 0 {
		reviewerID = &reviewerIDs[0]
	}
	var versionID *int64

	err := tx.QueryRow(ctx, `
		SELECT checklist_version_id
		FROM pa_requests
		WHERE id = $1
		  AND deleted_at IS NULL
	`, requestID).Scan(&versionID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("PA request not found")
		}
		return nil, fmt.Errorf("load PA request: %w", err)
	}

	if versionID == nil {
		return nil, fmt.Errorf("PA request has no checklist version assigned")
	}

	rows, err := tx.Query(ctx, `
		SELECT id, rule_key, description, required_field_key
		FROM checklist_rules
		WHERE checklist_version_id = $1
		  AND is_required = TRUE
		  AND required_field_key IS NOT NULL
		ORDER BY id
	`, *versionID)
	if err != nil {
		return nil, fmt.Errorf("load checklist rules: %w", err)
	}

	rules := make([]checklistRule, 0)
	for rows.Next() {
		var rule checklistRule
		if err := rows.Scan(
			&rule.ID,
			&rule.RuleKey,
			&rule.Description,
			&rule.FieldKey,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan checklist rule: %w", err)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate checklist rules: %w", err)
	}
	rows.Close()

	if len(rules) == 0 {
		return nil, fmt.Errorf("checklist version has no required field rules")
	}

	evaluation := &ChecklistEvaluation{
		PARequestID: requestID,
		Results:     make([]ChecklistResult, 0, len(rules)),
	}

	for _, rule := range rules {
		var field extractedField
		providerRole := providerNPIFieldRole(rule.FieldKey)
		err := tx.QueryRow(ctx, `
			SELECT field_value, verification_status, evidence_text
			FROM extracted_fields
			WHERE pa_request_id = $1
			  AND field_key = $2
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		`, requestID, rule.FieldKey).Scan(
			&field.Value,
			&field.VerificationStatus,
			&field.EvidenceText,
		)
		if providerRole != "" && (errors.Is(err, pgx.ErrNoRows) || (err == nil && field.VerificationStatus != "verified" && field.VerificationStatus != "corrected")) {
			var value string
			requestErr := tx.QueryRow(ctx, `
				SELECT CASE WHEN $2 = 'ordering' THEN COALESCE(NULLIF(ordering_provider_npi, ''), provider_npi, '')
				            ELSE COALESCE(servicing_provider_npi, '') END
				FROM pa_requests WHERE id=$1 AND deleted_at IS NULL
			`, requestID, providerRole).Scan(&value)
			if requestErr != nil {
				return nil, fmt.Errorf("load request provider NPI for checklist field %q: %w", rule.FieldKey, requestErr)
			}
			if strings.TrimSpace(value) != "" {
				evidence := "NPI entered in PA request as " + providerRole + " provider. Reviewer verification is still required."
				field.Value = &value
				field.VerificationStatus = "pending"
				field.EvidenceText = &evidence
				err = nil
			}
		}

		var status string
		var reason string
		var evidence *string

		switch {
		case err == pgx.ErrNoRows:
			status, reason, evidence = evaluateExtractedField(
				extractedField{},
				false,
			)

		case err != nil:
			return nil, fmt.Errorf(
				"load extracted field %q: %w",
				rule.FieldKey,
				err,
			)

		default:
			status, reason, evidence = evaluateExtractedField(
				field,
				true,
			)
		}

		var result ChecklistResult
		err = tx.QueryRow(ctx, `
			INSERT INTO checklist_results (
				pa_request_id,
				checklist_rule_id,
				status,
				reason,
				evidence,
				reviewed_by,
				reviewed_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, CASE WHEN $6::bigint IS NULL THEN NULL ELSE NOW() END, NOW())
			ON CONFLICT (pa_request_id, checklist_rule_id)
			DO UPDATE SET
				status = EXCLUDED.status,
				reason = EXCLUDED.reason,
				evidence = EXCLUDED.evidence,
				reviewed_by = EXCLUDED.reviewed_by,
				reviewed_at = EXCLUDED.reviewed_at,
				updated_at = NOW()
			RETURNING id, status, reason, evidence
		`, requestID, rule.ID, status, reason, evidence, reviewerID).Scan(
			&result.ID,
			&result.Status,
			&result.Reason,
			&result.Evidence,
		)
		if err != nil {
			return nil, fmt.Errorf("save checklist result: %w", err)
		}

		result.RuleID = rule.ID
		result.RuleKey = rule.RuleKey
		result.Description = rule.Description
		result.FieldKey = rule.FieldKey

		evaluation.Results = append(evaluation.Results, result)
		evaluation.Summary.Total++

		switch result.Status {
		case "fulfilled":
			evaluation.Summary.Fulfilled++
		case "missing":
			evaluation.Summary.Missing++
		case "needs_review":
			evaluation.Summary.NeedsReview++
		}
	}

	return evaluation, nil
}

func providerNPIFieldRole(fieldKey string) string {
	key := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(fieldKey)))
	switch key {
	case "provider_npi", "ordering_provider_npi":
		return "ordering"
	case "servicing_provider_npi":
		return "servicing"
	default:
		return ""
	}
}

func evaluateExtractedField(field extractedField, found bool) (string, string, *string) {
	if !found {
		return "missing", "Required field was not found in the extracted data.", nil
	}

	if field.Value == nil || strings.TrimSpace(*field.Value) == "" {
		return "missing", "Required field exists but its value is empty.", nil
	}

	evidence := field.EvidenceText
	if evidence == nil {
		evidence = field.Value
	}

	if field.VerificationStatus == "verified" ||
		field.VerificationStatus == "corrected" {
		return "fulfilled", "Required field has a non-empty value and has been verified.", evidence
	}

	return "needs_review", "Required field needs verification before it can be considered fulfilled.", evidence
}
