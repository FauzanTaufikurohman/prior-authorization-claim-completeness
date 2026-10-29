import type { Metadata } from "next";
import { DM_Sans, Geist_Mono, Manrope } from "next/font/google";
import "./globals.css";
import "./landing-page.css";

const dmSans = DM_Sans({
  variable: "--font-dm-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

const manrope = Manrope({
  variable: "--font-manrope",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Ayuprama | Prior Authorization dan Kelengkapan Klaim",
  description: "Ruang kerja untuk menyiapkan permohonan prior authorization dan meninjau kelengkapan klaim.",
  icons: {
    icon: "/images/logo-pa.jpeg",
    apple: "/images/logo-pa.jpeg",
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="id" className={`${dmSans.variable} ${manrope.variable} ${geistMono.variable} h-full antialiased`}>
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
