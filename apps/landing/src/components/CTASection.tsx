import { LuArrowUpRight } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import { site } from "@/lib/site";

export default function CTASection() {
  return (
    <section className="cta-section">
      <div className="container">
        <div className="cta-inner">
          <div>
            <p className="eyebrow">BUILD WITH THE BOUNDARY IN MIND</p>
            <h2>
              Give private data
              <br />a considered home.
            </h2>
            <p>Explore the architecture. Read the implementation. Make it yours.</p>
          </div>
          <div className="cta-actions">
            <a href={site.quickStart} className="button">
              Start with the quick start <LuArrowUpRight />
            </a>
            <a href={site.repo} className="text-link">
              <SiGithub /> Explore Gardbase on GitHub <LuArrowUpRight />
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}
