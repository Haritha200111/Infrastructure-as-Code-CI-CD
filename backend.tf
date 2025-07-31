terraform {
  backend "s3" {
    bucket         = "terra-state-bucket-haritha"
    key            = "dev/terraform.tfstate"
    region         = "us-east-2"
    encrypt        = true
    dynamodb_table = "terraform-lock-table"
  }
}
