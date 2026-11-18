import { LuArrowUpRight, LuCheck, LuServer } from "react-icons/lu";
import { site } from "@/lib/site";

export default function DeploymentSection() {
  return (
    <section id="deploy" className="section deployment-section">
      <div className="deployment-layout container">
        <div>
          <p className="eyebrow">05 / YOUR INFRASTRUCTURE</p>
          <h2>
            Run it in
            <br />
            your AWS account.
          </h2>
          <p className="deployment-description">
            Terraform configuration and Docker build targets live alongside the application. Review
            the deployment guide, configure your environment, and operate the stack on your terms.
          </p>
          <a href={site.deployment} className="text-link">
            Read the deployment guide <LuArrowUpRight />
          </a>
        </div>
        <div className="deployment-card">
          <div className="deployment-card-top">
            <LuServer />
            <span className="mono">SELF-HOSTED</span>
            <span className="license-tag">Apache 2.0</span>
          </div>
          <h3>
            Open-source software.
            <br />
            <span>Your cloud bill.</span>
          </h3>
          <p>
            The source is available under Apache 2.0. AWS infrastructure and operations are your
            responsibility.
          </p>
          <ul>
            {[
              "API and enclave service source included",
              "Terraform for EC2, KMS, DynamoDB, and S3",
              "Configurable region and enclave resources",
            ].map(item => (
              <li key={item}>
                <LuCheck />
                {item}
              </li>
            ))}
          </ul>
          <a className="button" href={site.deployment}>
            Plan your deployment <LuArrowUpRight />
          </a>
        </div>
      </div>
    </section>
  );
}
