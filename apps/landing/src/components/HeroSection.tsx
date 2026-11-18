import { LuArrowDown, LuArrowUpRight, LuCpu, LuDatabase, LuCode, LuScale } from "react-icons/lu";
import { SiGithub } from "react-icons/si";
import EncryptionDemo from "@/components/EncryptionDemo";
import { site } from "@/lib/site";

const foundations = [
  { icon: LuCpu, title: "AWS Nitro", text: "Isolated key operations" },
  { icon: LuDatabase, title: "DynamoDB + S3", text: "Encrypted object storage" },
  { icon: LuCode, title: "Go + REST", text: "Build your integration" },
  { icon: LuScale, title: "Apache 2.0", text: "Self-host in your AWS account" },
];

export default function HeroSection() {
  return (
    <section className="hero">
      <div className="hero-backdrop" aria-hidden="true" />
      <div className="hero-layout container">
        <div className="hero-copy">
          <a className="hero-eyebrow" href={site.licenseUrl}>
            <span className="status-dot" />
            Open source · Encrypted by design
            <LuArrowUpRight />
          </a>
          <h1>
            Keep sensitive data <span>on your side.</span>
          </h1>
          <p className="hero-description">
            An open-source encrypted data layer for AWS. Encrypt objects in your application, store
            them in DynamoDB and S3, and isolate key operations in Nitro Enclaves.
          </p>
          <div className="button-row">
            <a href={site.docs} className="button">
              Explore the documentation <LuArrowUpRight />
            </a>
            <a href={site.repo} className="button button-secondary">
              <SiGithub /> View source
            </a>
          </div>
          <a href="#how-it-works" className="text-link hero-detail">
            Follow the data <LuArrowDown />
          </a>
        </div>
        <EncryptionDemo />
      </div>
      <div className="container">
        <div className="foundation-strip" aria-label="Project foundations">
          <p className="foundation-label">
            A familiar stack.
            <strong>A different trust boundary.</strong>
          </p>
          <ul>
            {foundations.map(item => (
              <li key={item.title}>
                <item.icon aria-hidden="true" />
                <span>
                  <b>{item.title}</b>
                  <small>{item.text}</small>
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
