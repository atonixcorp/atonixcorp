// app/layout.tsx

import "./styles/base.css";
import "./styles/theme.css";
import "./styles/layout.css";

export const metadata = {
  title: "AtonixCorp Platform",
  description: "Autonomous cloud, HPC, networking, and secure infrastructure.",
  icons: {
    icon: "/atonixcorp-icon.png",
    shortcut: "/atonixcorp-icon.png",
    apple: "/atonixcorp-icon.png",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
      </body>
    </html>
  );
}
