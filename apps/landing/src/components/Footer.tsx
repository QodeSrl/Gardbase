import React from "react";
import LogoWhite from "../assets/logo-white.svg";

const footerCols = [
  {
    name: "Product",
    links: [
      { name: "Features", href: "#features" },
      { name: "Pricing", href: "#pricing" },
      { name: "Documentation", href: "#" },
      { name: "API Reference", href: "#" },
    ],
  },
  {
    name: "Company",
    links: [
      { name: "About", href: "https://qodesrl.com/" },
      { name: "Blog", href: "#" },
      { name: "Careers", href: "#" },
      { name: "Contact", href: "https://qodesrl.com/" },
    ],
  },
  {
    name: "Legal",
    links: [
      { name: "Privacy Policy", href: "#" },
      { name: "Terms of Service", href: "#" },
      { name: "Security", href: "#" },
      { name: "Compliance", href: "#" },
    ],
  },
];

const Footer: React.FC = () => {
  /* All links will be subject to change here */

  return (
    <footer className="bg-brand py-8 text-white sm:py-12">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid gap-8 sm:grid-cols-2 md:grid-cols-4">
          <div className="sm:col-span-2 md:col-span-1">
            <div className="mb-4 flex items-center space-x-2">
              <img draggable={false} src={LogoWhite} alt="Gardbase Logo" className="h-6 sm:h-8" />
            </div>
            <p className="text-sm text-gray-400 sm:text-base">
              The GDPR-native database that protects by design.
            </p>
          </div>

          {footerCols.map((col, index) => (
            <div key={index}>
              <h3 className="mb-3 text-sm font-semibold sm:mb-4 sm:text-base">{col.name}</h3>
              <ul className="space-y-2 text-xs text-gray-400 sm:text-sm">
                {col.links.map((link, linkIndex) => (
                  <li key={linkIndex}>
                    <a href={link.href} className="transition-colors hover:text-white">
                      {link.name}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-8 border-t border-gray-800 pt-6 text-center text-xs text-gray-400 sm:pt-8 sm:text-sm">
          <p>&copy; 2025 QodeSrl - Gardbase. All rights reserved.</p>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
