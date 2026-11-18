import { useEffect, useRef, useState } from "react";
import { LuArrowUpRight, LuMenu, LuMoon, LuSun, LuX } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import Logo from "../assets/logo.svg";
import LogoWhite from "../assets/logo-white.svg";
import { site } from "@/lib/site";
import { useTheme } from "@/lib/themeContext";

const links = [
  { label: "Architecture", href: "#how-it-works" },
  { label: "Capabilities", href: "#features" },
  { label: "Developers", href: "#open-source" },
  { label: "Deploy", href: "#deploy" },
];

export default function Header() {
  const { theme, toggle } = useTheme();
  const [open, setOpen] = useState(false);
  const menuButton = useRef<HTMLButtonElement>(null);
  const header = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!open) return;
    const dismiss = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        menuButton.current?.focus();
      }
    };
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !header.current?.contains(event.target)) setOpen(false);
    };
    document.addEventListener("keydown", dismiss);
    document.addEventListener("pointerdown", outside);
    return () => {
      document.removeEventListener("keydown", dismiss);
      document.removeEventListener("pointerdown", outside);
    };
  }, [open]);

  return (
    <header className="site-header" ref={header}>
      <div className="header-inner container">
        <a href="#top" className="brand" aria-label="Gardbase home" onClick={() => setOpen(false)}>
          <img src={theme === "dark" ? LogoWhite : Logo} alt="Gardbase" width="156" height="32" />
        </a>
        <nav className="desktop-nav" aria-label="Main navigation">
          {links.map(link => (
            <a key={link.href} href={link.href}>
              {link.label}
            </a>
          ))}
        </nav>
        <div className="header-actions">
          <button
            className="icon-button"
            onClick={toggle}
            aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
          >
            {theme === "dark" ? <LuSun /> : <LuMoon />}
          </button>
          <a href={site.repo} className="icon-button github-nav" aria-label="Gardbase on GitHub">
            <SiGithub />
          </a>
          <a href={site.docs} className="button button-small header-cta">
            Read the docs <LuArrowUpRight />
          </a>
          <button
            ref={menuButton}
            className="icon-button menu-toggle"
            aria-label={open ? "Close navigation" : "Open navigation"}
            aria-expanded={open}
            aria-controls="mobile-navigation"
            onClick={() => setOpen(!open)}
          >
            {open ? <LuX /> : <LuMenu />}
          </button>
        </div>
      </div>
      <nav
        id="mobile-navigation"
        className="mobile-nav container"
        aria-label="Mobile navigation"
        hidden={!open}
      >
        {links.map(link => (
          <a key={link.href} href={link.href} onClick={() => setOpen(false)}>
            {link.label}
            <LuArrowUpRight />
          </a>
        ))}
        <a href={site.docs}>
          Documentation <LuArrowUpRight />
        </a>
        <a href={site.repo}>
          GitHub <SiGithub />
        </a>
      </nav>
    </header>
  );
}
