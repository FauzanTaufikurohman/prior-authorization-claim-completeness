package services

import "testing"

func stringPointer(value string) *string {
	return &value
}

func TestEvaluateExtractedField(t *testing.T) {
	tests := []struct {
		name         string
		field        extractedField
		found        bool
		wantStatus   string
		wantEvidence *string
	}{
		{
			name:       "field not found",
			found:      false,
			wantStatus: "missing",
		},
		{
			name: "nil value",
			field: extractedField{
				Value: nil,
			},
			found:      true,
			wantStatus: "missing",
		},
		{
			name: "empty value",
			field: extractedField{
				Value: stringPointer(""),
			},
			found:      true,
			wantStatus: "missing",
		},
		{
			name: "whitespace value",
			field: extractedField{
				Value: stringPointer("   "),
			},
			found:      true,
			wantStatus: "missing",
		},
		{
			name: "verified field",
			field: extractedField{
				Value:              stringPointer("JKN"),
				VerificationStatus: "verified",
			},
			found:        true,
			wantStatus:   "fulfilled",
			wantEvidence: stringPointer("JKN"),
		},
		{
			name: "corrected field",
			field: extractedField{
				Value:              stringPointer("ICD-10 A09"),
				VerificationStatus: "corrected",
			},
			found:        true,
			wantStatus:   "fulfilled",
			wantEvidence: stringPointer("ICD-10 A09"),
		},
		{
			name: "pending field",
			field: extractedField{
				Value:              stringPointer("JKN"),
				VerificationStatus: "pending",
			},
			found:        true,
			wantStatus:   "needs_review",
			wantEvidence: stringPointer("JKN"),
		},
		{
			name: "uncertain field",
			field: extractedField{
				Value:              stringPointer("A09"),
				VerificationStatus: "uncertain",
			},
			found:        true,
			wantStatus:   "needs_review",
			wantEvidence: stringPointer("A09"),
		},
		{
			name: "explicit evidence",
			field: extractedField{
				Value:              stringPointer("JKN"),
				VerificationStatus: "verified",
				EvidenceText:       stringPointer("Coverage: JKN"),
			},
			found:        true,
			wantStatus:   "fulfilled",
			wantEvidence: stringPointer("Coverage: JKN"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, evidence := evaluateExtractedField(
				tt.field,
				tt.found,
			)

			if status != tt.wantStatus {
				t.Errorf(
					"status = %q, want %q",
					status,
					tt.wantStatus,
				)
			}

			if tt.wantEvidence == nil {
				if evidence != nil {
					t.Errorf(
						"evidence = %q, want nil",
						*evidence,
					)
				}
			} else {
				if evidence == nil {
					t.Fatal("evidence = nil, want non-nil")
				}

				if *evidence != *tt.wantEvidence {
					t.Errorf(
						"evidence = %q, want %q",
						*evidence,
						*tt.wantEvidence,
					)
				}
			}
		})
	}
}
