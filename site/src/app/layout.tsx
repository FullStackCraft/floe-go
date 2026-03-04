import type { Metadata } from "next";
import "@/styles/globals.css";

export const metadata: Metadata = {
  title: "floe - Go Options Analytics",
  description: "Server-side Go package for options analytics: Black-Scholes pricing, Greeks, IV surfaces, dealer exposures, implied PDFs, hedge flow, and IV vs RV tooling.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="bg-[#FAFAFA] text-black antialiased min-h-screen">
        {children}
      </body>
    </html>
  );
}
