# aws-inventory

![Go](https://img.shields.io/badge/Go-1.23.6+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

A CLI tool written in Go that generates a comprehensive inventory of AWS resources across one or more accounts, exporting results to a formatted Excel spreadsheet (`.xlsx`). All services are inventoried concurrently for fast execution, and output can optionally be uploaded directly to S3.

---

## Supported Services

| Service | Excel Sheet(s) |
|---|---|
| EC2 | EC2, AMI, EBS, ELB |
| EKS | EKS |
| RDS | RDS |
| Lambda | Lambda |
| IAM | Roles, Policies, Users, Groups |
| S3 | S3 |
| VPC | VPC, Subnets, Route Tables, NACLs, Security Groups |
| DynamoDB | DynamoDB |
| API Gateway | APIGW |
| CloudWatch Logs | Cloudwatch |
| ACM | ACM |
| EFS | EFS |
| EMR | EMR |
| OpenSearch | Opensearch |
| Route 53 | Route53 |
| SQS | SQS |
| SNS | SNS |
| WorkSpaces | Workspaces |

---

## Requirements

- Go 1.23.6+
- AWS credentials (access key + secret key) available as environment variables for each configured account

---

## Configuration

Copy and edit `config.yaml`. The configuration has three top-level sections:

**`environments`** — one entry per AWS account:

```yaml
environments:
  my-account-a:
    name: "Account A"
    skip_iam: false     # skip IAM inventory for this account
    skip_r53: false     # skip Route 53 inventory for this account
    credentials:
      access: "ACCOUNT_A_ACCESS_KEY"    # name of the env var holding the access key
      secret: "ACCOUNT_A_SECRET_KEY"    # name of the env var holding the secret key
    region: "us-east-1"
  my-account-b:
    name: "Account B"
    skip_iam: true
    skip_r53: false
    credentials:
      access: "ACCOUNT_B_ACCESS_KEY"
      secret: "ACCOUNT_B_SECRET_KEY"
    region: "us-west-2"
```

**`contents`** — global toggle for which services to inventory (applies across all accounts):

```yaml
contents:
  ec2: true
  emr: true
  rds: true
  workspaces: true
  route53: true
  acm: true
  dynamodb: true
  s3: true
  elb: true
  opensearch: true
  apigw: true
  ebs: true
  lambda: true
  iam: true
  efs: true
  eks: true
  vpc: true
  sqs: true
  sns: true
  cloudwatch: true
```

**`output`** — controls where the Excel file is saved:

```yaml
output:
  upload: false         # set to true to upload output to S3
  prefix: "INVENTORY"  # output filename prefix
  bucket_env: "ENV"     # environment key to use for the S3 upload
  bucket_name: "NAME"
  bucket_path: "/"
```

### Credentials

The `credentials.access` and `credentials.secret` fields are **environment variable names**, not the credential values themselves. At runtime the tool reads those env vars to get the actual keys:

```bash
export MY_ACCESS_KEY_ENV_VAR=AKIA...
export MY_SECRET_KEY_ENV_VAR=...
```

> If a credential env var is missing or empty, that environment is skipped. If the upload environment's credentials are missing, the file is saved locally only.

---

## Usage

Run directly:

```bash
go run inventory.go
```

Or build and run:

```bash
go build -o aws-inventory inventory.go
./aws-inventory
```

Output is written to `<PREFIX>-YYYY-MM-DD.xlsx` in the current directory. The spreadsheet includes a **Summary** sheet with resource counts, plus one sheet per enabled service.

---

## Project Structure

```
.
├── inventory.go         # Entry point
├── config.yaml          # Configuration
├── types/               # Shared types (Config, Env, ServiceInventory)
├── acmutil/             # ACM
├── apiutil/             # API Gateway
├── cloudwatchutil/      # CloudWatch Logs
├── dyanmodbutil/        # DynamoDB
├── ec2util/             # EC2, AMI, EBS, ELB, VPC, Subnets, Route Tables, NACLs, SGs
├── efsutil/             # EFS
├── eksutil/             # EKS
├── emrutil/             # EMR
├── iamutil/             # IAM Roles, Policies, Users, Groups
├── lambdautil/          # Lambda
├── osutil/              # OpenSearch
├── rdsutil/             # RDS
├── route53util/         # Route 53
├── s3util/              # S3
├── snsutil/             # SNS
├── sqsutil/             # SQS
└── wsutil/              # WorkSpaces
```

---

## License

MIT