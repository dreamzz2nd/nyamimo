import type { Metadata } from "next";
import { Bebas_Neue, Inter } from "next/font/google";
import "./globals.css";
import 'swiper/css';
import 'swiper/css/navigation';
import 'swiper/css/pagination';
import { Providers } from "@/components/Providers";

const bebasNeue = Bebas_Neue({
  variable: "--font-bebas-neue",
  subsets: ["latin"],
  weight: "400",
});

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Animekudesu",
  description: "Stream your favorite anime in high quality, completely free and without any ads. Enjoy a seamless and uninterrupted viewing experience.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
      <html lang="en">
        <body
          className={`${bebasNeue.variable} ${inter.variable} antialiased bg-gray-800 text-white font-sans`}
        >
          <Providers>
            {children}
          </Providers>
        </body>
      </html>
  );
}
