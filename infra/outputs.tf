output "app_url" {
  description = "Where the app is served."
  value       = "https://${aws_cloudfront_distribution.main.domain_name}"
}

output "database_endpoint" {
  value = aws_db_instance.postgres.address
}

# Add these as repository variables (Settings > Secrets and variables >
# Actions > Variables) and create an environment named after var.environment.
# None of them are secrets.
output "github_actions_variables" {
  value = {
    AWS_REGION                 = var.region
    AWS_DEPLOY_ROLE_ARN        = aws_iam_role.github_deploy.arn
    ECR_REPOSITORY_URL         = aws_ecr_repository.api.repository_url
    ECS_CLUSTER                = aws_ecs_cluster.main.name
    ECS_SERVICE                = aws_ecs_service.api.name
    ECS_TASK_FAMILY            = local.api_family
    ECS_SUBNETS                = join(",", module.vpc.private_subnets)
    ECS_SECURITY_GROUP         = aws_security_group.api.id
    WEB_BUCKET                 = aws_s3_bucket.web.bucket
    CLOUDFRONT_DISTRIBUTION_ID = aws_cloudfront_distribution.main.id
  }
}
