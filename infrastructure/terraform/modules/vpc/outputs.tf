# VPC module outputs
# Exposes essential network infrastructure resources to other modules

output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.main.id
}

output "vpc_cidr" {
  description = "CIDR block of the VPC"
  value       = aws_vpc.main.cidr_block
}

output "public_subnet_ids" {
  description = "List of IDs of public subnets"
  value       = aws_subnet.public[*].id
}

output "private_app_subnet_ids" {
  description = "List of IDs of private application subnets"
  value       = aws_subnet.private_app[*].id
}

output "private_data_subnet_ids" {
  description = "List of IDs of private data subnets"
  value       = aws_subnet.private_data[*].id
}

output "availability_zones" {
  description = "List of availability zones used"
  value       = var.availability_zones
}

output "nat_gateway_ids" {
  description = "List of IDs of NAT gateways"
  value       = aws_nat_gateway.nat[*].id
}

output "internet_gateway_id" {
  description = "ID of the internet gateway"
  value       = aws_internet_gateway.main.id
}

output "default_security_group_id" {
  description = "ID of the default security group for the VPC"
  value       = aws_security_group.default.id
}

output "public_route_table_id" {
  description = "ID of the public route table"
  value       = aws_route_table.public.id
}

output "private_app_route_table_ids" {
  description = "List of IDs of private application route tables"
  value       = aws_route_table.private_app[*].id
}

output "private_data_route_table_ids" {
  description = "List of IDs of private data route tables"
  value       = aws_route_table.private_data[*].id
}

output "s3_endpoint_id" {
  description = "ID of the S3 VPC endpoint if enabled"
  value       = var.enable_s3_endpoint ? aws_vpc_endpoint.s3[0].id : null
}