const repo = "https://github.com/QodeSrl/Gardbase";
const source = `${repo}/blob/development`;

export const site = {
  name: "Gardbase",
  repo,
  source,
  repoLabel: "QodeSrl/Gardbase",
  docs: `${source}/README.md`,
  quickStart: `${source}/README.md#quick-start`,
  security: `${source}/README.md#security-model`,
  deployment: `${source}/infrastructure/README.md`,
  apiDocs: `${source}/apps/api/README.md`,
  issues: `${repo}/issues`,
  license: "Apache 2.0",
  licenseUrl: `${repo}/blob/main/LICENSE`,
  company: "Qode",
  companyUrl: "https://qodesrl.com/",
  tagline: "Encrypted data infrastructure. Built in the open.",
  year: new Date().getFullYear(),
};
