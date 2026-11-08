variable "region" {
  description = "AWS region for everything except CloudFront, which is global."
  type        = string
  default     = "eu-west-1"
}

variable "environment" {
  description = "Short environment name used in resource names and tags."
  type        = string
  default     = "production"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,20}$", var.environment))
    error_message = "Use lowercase letters, digits and hyphens, starting with a letter."
  }
}

variable "github_repository" {
  description = "owner/name of the repository whose main branch may deploy."
  type        = string
  default     = "dehghanpur10/fairshare"
}

variable "create_github_oidc_provider" {
  description = "An account can only have one GitHub OIDC provider. Set false if it already exists."
  type        = bool
  default     = true
}

variable "vpc_cidr" {
  type    = string
  default = "10.40.0.0/16"
}

variable "single_nat_gateway" {
  description = "One shared NAT gateway instead of one per AZ. Cheaper, but an AZ outage cuts egress."
  type        = bool
  default     = true
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_engine_version" {
  description = "Major version of RDS for PostgreSQL. Minor versions upgrade automatically."
  type        = string
  default     = "18"
}

variable "db_multi_az" {
  type    = bool
  default = false
}

variable "db_deletion_protection" {
  type    = bool
  default = true
}

variable "api_cpu" {
  description = "Fargate CPU units for one API task (256 = 0.25 vCPU)."
  type        = number
  default     = 256
}

variable "api_memory" {
  description = "Fargate memory in MiB for one API task."
  type        = number
  default     = 512
}

variable "api_min_tasks" {
  type    = number
  default = 1
}

variable "api_max_tasks" {
  type    = number
  default = 4
}

variable "api_image_tag" {
  description = "Image tag for the first deploy only. CI registers new task definition revisions after that."
  type        = string
  default     = "bootstrap"
}

variable "log_retention_days" {
  type    = number
  default = 30
}
