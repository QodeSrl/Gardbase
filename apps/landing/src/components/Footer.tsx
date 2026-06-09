import React from "react";
import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";
import { useTheme } from "@/lib/themeContext";

const footerCols = [
  {
    name: "Product",
    links: [
      { name: "Why Gardbase", href: "#why" },
      { name: "Features", href: "#features" },
      { name: "How it Works", href: "#how-it-works" },
      { name: "Pricing", href: "#pricing" },
    ],
  },
  {
    name: "Open Source",
    links: [
      { name: "GitHub", href: site.repo, external: true },
      { name: "Documentation", href: site.docs, external: true },
      { name: `License (${site.license})`, href: site.licenseUrl, external: true },
      { name: "Contributing", href: `${site.repo}#contributing`, external: true },
    ],
  },
  {
    name: "Company",
    links: [
      { name: "Qode", href: site.companyUrl, external: true },
      { name: "Contact", href: site.companyUrl, external: true },
    ],
  },
];

const Footer: React.FC = () => {
  const { theme } = useTheme();
  return (
    <footer className="border-t border-line bg-bg2 py-12">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid gap-10 sm:grid-cols-2 md:grid-cols-4">
          <div className="sm:col-span-2 md:col-span-1">
            <img
              draggable={false}
              src={theme === "dark" ? LogoWhite : Logo}
              alt="Gardbase"
              className="h-7"
            />
            <p className="mt-4 max-w-xs text-sm text-subtle">{site.tagline}</p>
            <a
              href={site.repo}
              target="_blank"
              rel="noreferrer"
              className="mt-4 inline-flex items-center gap-2 rounded-lg border border-line bg-fg/5 px-3 py-2 text-sm text-muted transition-colors hover:border-fg/30 hover:bg-fg/10 hover:text-fg"
            >
              <SiGithub className="h-4 w-4" /> {site.repoLabel}
            </a>
          </div>

          {footerCols.map(col => (
            <div key={col.name}>
              <h3 className="mb-4 text-sm font-semibold text-fg">{col.name}</h3>
              <ul className="space-y-2.5 text-sm text-subtle">
                {col.links.map(link => (
                  <li key={link.name}>
                    <a
                      href={link.href}
                      {...("external" in link && link.external
                        ? { target: "_blank", rel: "noreferrer" }
                        : {})}
                      className="transition-colors hover:text-fg"
                    >
                      {link.name}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-10 flex flex-col items-center justify-between gap-3 border-t border-line pt-6 text-center text-xs text-subtle sm:flex-row sm:text-left sm:text-sm">
          <p>
            &copy; {site.year} {site.company} · Gardbase. Released under the {site.license} license.
          </p>
          <p>Encrypted by design. Open by default.</p>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
