package models

import "time"

type PARequest struct {
	ID                 int64      `json:"id"`
	RequestNumber      string     `json:"request_number"`
	CreatedBy          int64      `json:"created_by"`
	PayerName          string     `json:"payer_name"`
	ServiceType        string     `json:"service_type"`
	ProviderNPI        string     `json:"provider_npi"`
	PatientReference   string     `json:"patient_reference"`
	Diagnosis          string     `json:"diagnosis"`
	Status             string     `json:"status"`
	ChecklistVersionID *int64     `json:"checklist_version_id"`
	PayerReference     *string    `json:"payer_reference"`
	SubmittedAt        *time.Time `json:"submitted_at"`
	PayerDecision      *string    `json:"payer_decision"`
	PayerDecisionAt    *time.Time `json:"payer_decision_at"`
	PayerDecisionNote  *string    `json:"payer_decision_note"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreatePARequestInput struct {
	PayerName          string `json:"payer_name" binding:"required,max=150"`
	ServiceType        string `json:"service_type" binding:"required,max=100"`
	ProviderNPI        string `json:"provider_npi" binding:"omitempty,len=10,numeric"`
	PatientReference   string `json:"patient_reference" binding:"required,max=100"`
	Diagnosis          string `json:"diagnosis" binding:"required,max=2000"`
	ChecklistVersionID *int64 `json:"checklist_version_id"`
}

type UpdatePARequestInput struct {
	PayerName          string `json:"payer_name" binding:"required,max=150"`
	ServiceType        string `json:"service_type" binding:"required,max=100"`
	ProviderNPI        string `json:"provider_npi" binding:"omitempty,len=10,numeric"`
	PatientReference   string `json:"patient_reference" binding:"required,max=100"`
	Diagnosis          string `json:"diagnosis" binding:"required,max=2000"`
	ChecklistVersionID *int64 `json:"checklist_version_id"`
}
