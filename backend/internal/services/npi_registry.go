package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const npiRegistryURL = "https://npiregistry.cms.hhs.gov/api/"

var ErrNPIRecordNotFound = errors.New("NPI record not found")

type NPIProvider struct {
	Number           string `json:"number"`
	EnumerationType  string `json:"enumeration_type"`
	Status           string `json:"status"`
	Name             string `json:"name"`
	PracticeAddress  string `json:"practice_address,omitempty"`
	City             string `json:"city,omitempty"`
	State            string `json:"state,omitempty"`
	PostalCode       string `json:"postal_code,omitempty"`
	TaxonomyCode     string `json:"taxonomy_code,omitempty"`
	TaxonomyDesc     string `json:"taxonomy_description,omitempty"`
	RegistrySource   string `json:"registry_source"`
	RegistryNotice   string `json:"registry_notice"`
}

type npiRegistryResponse struct {
	Results []struct {
		Number          json.Number `json:"number"`
		EnumerationType string `json:"enumeration_type"`
		Basic           struct {
			FirstName        string `json:"first_name"`
			MiddleName       string `json:"middle_name"`
			LastName         string `json:"last_name"`
			NamePrefix       string `json:"name_prefix"`
			NameSuffix       string `json:"name_suffix"`
			Credential       string `json:"credential"`
			OrganizationName string `json:"organization_name"`
			Status           string `json:"status"`
			NPIStatus        string `json:"npi_status"`
		} `json:"basic"`
		Addresses []struct {
			Purpose     string `json:"address_purpose"`
			Address1    string `json:"address_1"`
			Address2    string `json:"address_2"`
			City        string `json:"city"`
			State       string `json:"state"`
			PostalCode  string `json:"postal_code"`
		} `json:"addresses"`
		Taxonomies []struct {
			Code    string `json:"code"`
			Desc    string `json:"desc"`
			Primary bool   `json:"primary"`
		} `json:"taxonomies"`
	} `json:"results"`
}

func LookupNPI(ctx context.Context, number string) (NPIProvider, error) {
	query := url.Values{}
	query.Set("version", "2.1")
	query.Set("number", number)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		npiRegistryURL+"?"+query.Encode(),
		nil,
	)
	if err != nil {
		return NPIProvider{}, fmt.Errorf("create NPI Registry request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return NPIProvider{}, fmt.Errorf("request NPI Registry: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return NPIProvider{}, fmt.Errorf("NPI Registry returned HTTP %d", response.StatusCode)
	}

	var registryData npiRegistryResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&registryData); err != nil {
		return NPIProvider{}, fmt.Errorf("decode NPI Registry response: %w", err)
	}
	if len(registryData.Results) == 0 {
		return NPIProvider{}, ErrNPIRecordNotFound
	}

	result := registryData.Results[0]
	name := strings.Join(strings.Fields(strings.Join([]string{
		result.Basic.NamePrefix,
		result.Basic.FirstName,
		result.Basic.MiddleName,
		result.Basic.LastName,
		result.Basic.NameSuffix,
		result.Basic.Credential,
	}, " ")), " ")
	if result.Basic.OrganizationName != "" {
		name = strings.TrimSpace(result.Basic.OrganizationName)
	}

	provider := NPIProvider{
		Number:          result.Number.String(),
		EnumerationType: result.EnumerationType,
		Status:          result.Basic.NPIStatus,
		Name:            name,
		RegistrySource:  npiRegistryURL,
		RegistryNotice:  "NPI tidak memastikan lisensi, credentialing, cakupan, atau ketentuan PA.",
	}
	if provider.Status == "" {
		provider.Status = result.Basic.Status
	}
	switch provider.Status {
	case "A":
		provider.Status = "Active"
	case "D":
		provider.Status = "Deactivated"
	}

	for _, address := range result.Addresses {
		if !strings.EqualFold(address.Purpose, "LOCATION") {
			continue
		}
		provider.PracticeAddress = strings.TrimSpace(strings.Join([]string{
			address.Address1,
			address.Address2,
		}, " "))
		provider.City = address.City
		provider.State = address.State
		provider.PostalCode = address.PostalCode
		break
	}

	for _, taxonomy := range result.Taxonomies {
		if taxonomy.Primary {
			provider.TaxonomyCode = taxonomy.Code
			provider.TaxonomyDesc = taxonomy.Desc
			break
		}
	}
	return provider, nil
}
