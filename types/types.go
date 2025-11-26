package types

type Env struct {
	Name        string            `yaml:"name"`
	Credentials map[string]string `yaml:"credentials"`
	Skip_iam    bool              `yaml:"skip_iam"`
	Skip_r53    bool              `yaml:"skip_r53"`
}

type Outfile struct {
	Prefix     string `yaml:"prefix"`
	BucketName string `yaml:"bucket_name"`
	BucketEnv  string `yaml:"bucket_env"`
	BucketPath string `yaml:"bucket_path"`
	Upload     bool   `yaml:"upload"`
}
type Contents struct {
	EC2        bool `yaml:"ec2"`
	EMR        bool `yaml:"emr"`
	RDS        bool `yaml:"rds"`
	WORKSPACES bool `yaml:"workspaces"`
	ROUTE53    bool `yaml:"route53"`
	ACM        bool `yaml:"acm"`
	DYNAMODB   bool `yaml:"dynamodb"`
	S3         bool `yaml:"s3"`
	ELB        bool `yaml:"elb"`
	OPENSEARCH bool `yaml:"opensearch"`
	APIGW      bool `yaml:"apigw"`
	EBS        bool `yaml:"ebs"`
	LAMBDA     bool `yaml:"lambda"`
	IAM        bool `yaml:"iam"`
	EFS        bool `yaml:"efs"`
	EKS        bool `yaml:"eks"`
	VPC        bool `yaml:"vpc"`
	SQS        bool `yaml:"sqs"`
	SNS        bool `yaml:"sns"`
	CLOUDWATCH bool `yaml:"cloudwatch"`
}
type Config struct {
	Environments map[string]Env `yaml:"environments"`
	Contents     Contents       `yaml:"contents"`
	Output       Outfile        `yaml:"output"`
}

type ServiceInventory struct {
	Name    string
	Channel chan [][]string
	Values  [][]string
	Message string
}
