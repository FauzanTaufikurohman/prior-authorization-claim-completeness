package handlers

import "testing"

func TestValidRequestStatus(t *testing.T) {
	valid := []string{
		"draft",
		"processing",
		"in_review",
		"ready_for_review",
		"completed",
		"error",
	}

	for _, status := range valid {
		if !validRequestStatus(status) {
			t.Errorf("expected status %q to be valid", status)
		}
	}

	if validRequestStatus("approved") {
		t.Error("approved is not a supported PA request status")
	}
}

func TestValidInput(t *testing.T) {
	tests := []struct {
		name      string
		payer     string
		service   string
		npi       string
		patient   string
		diagnosis string
		want      bool
	}{
		{
			name:      "valid input without NPI",
			payer:     "Example Payer",
			service:   "MRI",
			patient:   "SYNTHETIC-001",
			diagnosis: "Synthetic test diagnosis",
			want:      true,
		},
		{
			name:      "valid 10 digit NPI",
			payer:     "Example Payer",
			service:   "MRI",
			npi:       "1234567890",
			patient:   "SYNTHETIC-002",
			diagnosis: "Synthetic test diagnosis",
			want:      true,
		},
		{
			name:      "missing payer",
			service:   "MRI",
			patient:   "SYNTHETIC-003",
			diagnosis: "Synthetic test diagnosis",
			want:      false,
		},
		{
			name:      "invalid NPI length",
			payer:     "Example Payer",
			service:   "MRI",
			npi:       "12345",
			patient:   "SYNTHETIC-004",
			diagnosis: "Synthetic test diagnosis",
			want:      false,
		},
		{
			name:      "non numeric NPI",
			payer:     "Example Payer",
			service:   "MRI",
			npi:       "123456789A",
			patient:   "SYNTHETIC-005",
			diagnosis: "Synthetic test diagnosis",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validInput(
				tt.payer,
				tt.service,
				tt.npi,
				tt.patient,
				tt.diagnosis,
			)

			if got != tt.want {
				t.Errorf("validInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{input: "1", want: 1},
		{input: "25", want: 25},
		{input: "0", wantErr: true},
		{input: "-1", wantErr: true},
		{input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parsePositiveInt(tt.input)

		if tt.wantErr {
			if err == nil {
				t.Errorf("parsePositiveInt(%q) expected an error", tt.input)
			}
			continue
		}

		if err != nil || got != tt.want {
			t.Errorf("parsePositiveInt(%q) = %d, %v; want %d, nil",
				tt.input, got, err, tt.want)
		}
	}
}
