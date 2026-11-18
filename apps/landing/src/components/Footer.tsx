import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";
import { LuArrowUpRight } from "react-icons/lu";
import { site } from "@/lib/site";
import { useTheme } from "@/lib/themeContext";

const groups = [
  {
    title: "Explore",
    links: [
      { label: "Architecture", href: "#how-it-works" },
      { label: "Capabilities", href: "#features" },
      { label: "Deployment", href: "#deploy" },
      { label: "Questions", href: "#faq" },
    ],
  },
  {
    title: "Build",
    links: [
      { label: "Documentation", href: site.docs },
      { label: "API reference", href: site.apiDocs },
      { label: "Security model", href: site.security },
      { label: "GitHub issues", href: site.issues },
    ],
  },
  {
    title: "Project",
    links: [
      { label: "Source code", href: site.repo },
      { label: "Apache 2.0", href: site.licenseUrl },
      { label: "Contributing", href: `${site.docs}#contributing` },
      { label: "Qode", href: site.companyUrl },
    ],
  },
];

export default function Footer() {
  const { theme } = useTheme();
  return (
    <footer className="site-footer">
      <div className="container">
        <div className="footer-layout">
          <div className="footer-brand">
            <a href="#top" aria-label="Gardbase home">
              <img
                src={theme === "dark" ? LogoWhite : Logo}
                alt="Gardbase"
                width="156"
                height="32"
              />
            </a>
            <p>{site.tagline}</p>
            <span className="mono">YOUR DATA. YOUR AWS ACCOUNT.</span>
          </div>
          {groups.map(group => (
            <nav key={group.title} aria-label={group.title}>
              <h3>{group.title}</h3>
              {group.links.map(link => (
                <a key={link.label} href={link.href}>
                  {link.label}
                  {link.href.startsWith("https:") && <LuArrowUpRight />}
                </a>
              ))}
            </nav>
          ))}
        </div>
        <div className="footer-bottom">
          <span>
            © {site.year} {site.company}. Gardbase is licensed under Apache 2.0.
          </span>
          <span>Encrypted by design. Open to inspection.</span>
        </div>
      </div>
    </footer>
  );
}
