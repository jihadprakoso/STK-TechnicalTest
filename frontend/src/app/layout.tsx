import type { Metadata } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import "./globals.css";

const plusJakartaSans = Plus_Jakarta_Sans({
  subsets: ["latin"],
  variable: "--font-plus-jakarta-sans",
  display: "swap",
});

export const metadata: Metadata = {
  title: "System Management | Solusi Teknologi Kreatif",
  description: "Kelola hierarki menu dan navigasi aplikasi.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="id" className={plusJakartaSans.variable}>
      <body>{children}</body>
    </html>
  );
}
