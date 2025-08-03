## IaC CI/CD Pipeline Overview & Flow

- **Infrastructure as Code (IaC):** Managing and provisioning infrastructure (servers, networks, etc.) using code, rather than manual processes. This makes infrastructure changes versioned, repeatable, and automatable.

- **Terraform:** An open-source IaC tool that lets you define cloud and on-prem resources in configuration files. It supports many providers (AWS, Azure, GCP).

### How is Everything Configured & Run?
- **Configuration:** You write Terraform files (`.tf`) describing your desired infrastructure.
- **CI/CD Integration:** GitHub Actions (or Jenkins) automates the workflow when you push changes to your repo.

### The Flow
1. **Code Commit:** You update Terraform files and push to GitHub.
2. **Syntax Validation:** The pipeline checks for syntax errors to prevent broken code.
3. **Format & Validate:** Runs `terraform fmt` (auto-format) and `terraform validate` (checks for correctness).
4. **Plan:** Runs `terraform plan` to show what will change if applied.
5. **Manual Approval:** Someone reviews the plan and approves if changes are safe.
6. **Apply:** Runs `terraform apply` to make the changes live.
7. **State Storage:** Terraform state (current infra snapshot) is stored securely in a remote backend (like AWS S3).

### Why Use This Flow?
- **Consistency:** Same process every time.
- **Safety:** Manual approval prevents accidental changes.
- **Auditability:** All changes are tracked in version control.
- **Security:** State is stored securely and not lost.
