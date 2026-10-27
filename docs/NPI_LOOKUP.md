# CMS NPI Registry lookup

The authenticated endpoint `GET /api/providers/npi/{number}` looks up one
10-digit provider NPI in the official CMS NPPES NPI Registry API v2.1.

The response includes the registry number, enumeration type, registry status,
provider name, primary practice location, primary taxonomy, and source notice.
The create-request form uses this lookup to help staff identify the provider;
it still stores the NPI on the PA request and leaves the rest of the workflow
available when the registry cannot be reached.

Registry data is identification data only. An NPI does not establish licensure,
credentialing, coverage, clinical eligibility, or payer prior-authorization
requirements. Staff must review the returned record and use the applicable
payer policy for the checklist.

The API uses one direct lookup per user action. It does not perform bulk
searches or retain a separate copy of provider data.
