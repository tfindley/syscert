# Code of Conduct

## Part 1 — Contributor Covenant

### Our Pledge

We as contributors and maintainers pledge to make participation in this project a harassment-free experience for everyone, regardless of age, body size, visible or invisible disability, ethnicity, sex characteristics, gender identity and expression, level of experience, education, socioeconomic status, nationality, personal appearance, race, caste, colour, religion, or sexual identity and orientation.

We pledge to act and interact in ways that contribute to an open, welcoming, diverse, inclusive, and healthy community.

### Our Standards

Examples of behaviour that contributes to a positive environment:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

Examples of unacceptable behaviour:

- The use of sexualised language or imagery, and sexual attention or advances of any kind
- Trolling, insulting or derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information, such as a physical or electronic address, without explicit permission
- Other conduct which could reasonably be considered inappropriate in a professional setting

### Enforcement Responsibilities

Maintainers are responsible for clarifying and enforcing acceptable standards of behaviour and will take appropriate and fair corrective action in response to any behaviour they deem inappropriate, threatening, offensive, or harmful.

Maintainers have the right and responsibility to remove, edit, or reject comments, commits, code, issues, and other contributions that are not aligned with this Code of Conduct, and will communicate reasons for moderation decisions when appropriate.

### Scope

This Code of Conduct applies within all project spaces, and also applies when an individual is officially representing the project in public spaces.

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behaviour may be reported by opening an issue on the project's GitHub repository. All complaints will be reviewed and investigated promptly and fairly.

Maintainers who do not follow or enforce the Code of Conduct in good faith may face temporary or permanent repercussions as determined by other members of the project's leadership.

### Attribution

This section is adapted from the [Contributor Covenant](https://www.contributor-covenant.org), version 2.1.

---

## Part 2 — Acceptable Use Policy

This software is provided as a diagnostic tool for legitimately testing and understanding OIDC/OAuth2 SSO configurations. Its use is subject to the following conditions.

### Permitted use

- Diagnosing and debugging your own SSO configuration
- Testing OIDC providers on systems you own or administer
- Security research on systems for which you have explicit written authorisation
- Educational and demonstration purposes, with honest and transparent disclosure to any users

### Prohibited use

Use of this software for any of the following purposes is explicitly prohibited:

1. **Credential harvesting** — Deploying an instance of this tool to deceive users into authenticating with the intent to capture, store, reuse, or sell their tokens, credentials, or personal data.

2. **Unauthorised access** — Using this tool against OIDC providers or protected resources without the explicit permission of the system owner.

3. **Deceptive impersonation** — Presenting this tool as a legitimate service of a third party (e.g. a bank, employer, or other organisation) to trick users into authenticating.

4. **Non-consensual data collection** — Collecting or retaining user identity data (tokens, claims, profile information) without the informed consent of the users whose data is being processed.

5. **Surveillance or tracking** — Using the tool or its output to monitor, profile, or track individuals without their knowledge and consent.

6. **Facilitating other prohibited uses** — Providing this tool as a service, component, or dependency within a system whose primary purpose is one of the above.

### Responsibility

The operator of any deployed instance of this software is solely responsible for ensuring its use complies with applicable laws (including but not limited to GDPR, UK GDPR, CCPA, and local equivalents), the terms of service of any OIDC providers it is configured against, and this Acceptable Use Policy.

The authors of this software accept no liability for misuse. Misuse may violate computer fraud laws, data protection law, and the terms of service of identity providers. Violators may face civil and criminal liability.

### Reporting misuse

If you become aware of a deployed instance of this tool being used for any prohibited purpose, please report it by opening an issue at the project's GitHub repository, or by contacting the maintainer directly via GitHub.