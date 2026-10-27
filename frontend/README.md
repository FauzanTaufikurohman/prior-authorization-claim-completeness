This is a [Next.js](https://nextjs.org) project bootstrapped with [`create-next-app`](https://nextjs.org/docs/app/api-reference/cli/create-next-app).

## Prior Authorization Assistant

Copy `frontend/.env.example` to `frontend/.env.local` and
`backend/.env.example` to `backend/.env`. Set `BACKEND_API_URL` to the Go API,
for example `http://localhost:8080/api`, and fill in the database settings.
The Next.js rewrite proxies `/api` requests to the backend, so the browser
does not need cross-origin access.

Set `OPENAI_API_KEY` and optionally `OPENAI_MODEL` in the backend `.env` file.
The API key must only be configured on the backend; never add it to a
`NEXT_PUBLIC_*` variable or frontend source. PDF extraction sends extracted
text to OpenAI and therefore requires an explicit confirmation in the UI.
Use synthetic or de-identified documents for development and demos. Do not
process identifiable patient data until the required privacy, security, and
vendor-contract review is complete.

The MVP parses two-column CSVs as label/value pairs regardless of header names, wider CSV tables by their header labels, and text-based PDFs for `:` or `=` pairs and simple tables. Scanned PDFs are not supported.
Scanned PDFs and OCR are not supported. Every AI-extracted field starts in
pending review and must be verified by a person before checklist results are
treated as complete.

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.
