terraform {
  # 1.10: native S3 state locking (use_lockfile), so no DynamoDB table.
  # 1.11: write-only arguments, which keep the JWT secret out of state.
  required_version = ">= 1.11"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.72"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.9"
    }
  }

  # Partial configuration: pass the bucket at init time, see backend.hcl.example.
  backend "s3" {}
}

provider "aws" {
  region = var.region

  default_tags {
    tags = {
      Project     = "fairshare"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}
