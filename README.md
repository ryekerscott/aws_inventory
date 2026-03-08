# AWS Inventory

A Go tool that inventories AWS resources across multiple accounts and exports the results to a formatted Excel spreadsheet (.xlsx). Inventories run concurrently per service and can optionally upload the output file to S3.

## Supported Services

| Service | Sheet(s) |
|---|---|
| EC2 | EC2, AMI |
| EMR | EMR |
| RDS | RDS |
| WorkSpaces | Workspaces |
| Route 53 | Route53 |
| ACM | ACM |
| DynamoDB | DynamoDB |
| S3 | S3 |
| ELB | ELB |
| OpenSearch | Opensearch |
| API Gateway | APIGW |
| EBS | EBS |
| Lambda | Lambda |
| IAM | Roles, Policies, Users, Groups |
| EFS | EFS |
| EKS | EKS |
| VPC | VPC, Subnets, RTBs, NACLs, SGs |
| SQS | SQS |
| SNS | SNS |
| CloudWatch Logs | Cloudwatch |

## Requirements

- Go 1.23.6+
- AWS credentials (access key + secret) set as environment variables for each configured environment

## Configuration

Copy and edit `config.yaml`:

```yaml
environments:
  my-account:
    name: "My Account"
    credentials:
      access: "MY_ACCESS_KEY_ENV_VAR"   # name of the env var holding the access key
      secret: "MY_SECRET_KEY_ENV_VAR"   # name of the env var holding the secret key
      region: "us-east-1"

contents:
  ec2: true
  emr: false
  rds: true
  workspaces: false
  route53: true
  acm: true
  dynamodb: true
  s3: true
  elb: true
  opensearch: false
  apigw: true
  ebs: true
  lambda: true
  iam: true
  efs: false
  eks: false
  vpc: true
  sqs: true
  sns: true
  cloudwatch: true

output:
  upload: false          # set to true to upload the output file to S3
  prefix: "INVENTORY"   # output filename prefix
  bucket_env: "my-account"  # environment key to use for the S3 upload
  bucket_name: "my-bucket"
  bucket_path: "/"
```

### Credentials

The `credentials.access` and `credentials.secret` fields are **environment variable names**, not the credential values themselves. At runtime the tool reads those env vars to get the actual keys:

```sh
export MY_ACCESS_KEY_ENV_VAR=AKIA...
export MY_SECRET_KEY_ENV_VAR=...
```

If a credential env var is missing or empty, that environment is skipped. If the upload environment's credentials are missing, uploading is disabled and the file is saved locally only.

## Usage

```sh
go run inventory.go
```

Or build first:

```sh
go build -o aws-inventory inventory.go
./aws-inventory
```

The tool outputs a file named `<prefix>-YYYY-MM-DD.xlsx` in the current directory. The spreadsheet contains a **Summary** sheet with resource counts plus one sheet per enabled service.

## Project Structure

```
.
├── inventory.go          # Entry point
├── config.yaml           # Configuration file
├── types/                # Shared types (Config, Env, ServiceInventory)
├── acmutil/              # ACM inventory
├── apiutil/              # API Gateway inventory
├── cloudwatchutil/       # CloudWatch Logs inventory
├── dyanmodbutil/         # DynamoDB inventory
├── ec2util/              # EC2, AMI, EBS, ELB, VPC, Subnets, RTBs, NACLs, SGs
├── efsutil/              # EFS inventory
├── eksutil/              # EKS inventory
├── emrutil/              # EMR inventory
├── iamutil/              # IAM Roles, Policies, Users, Groups
├── lambdautil/           # Lambda inventory
├── osutil/               # OpenSearch inventory
├── rdsutil/              # RDS inventory
├── route53util/          # Route 53 inventory
├── s3util/               # S3 inventory
├── snsutil/              # SNS inventory
├── sqsutil/              # SQS inventory
└── wsutil/               # WorkSpaces inventory
```
