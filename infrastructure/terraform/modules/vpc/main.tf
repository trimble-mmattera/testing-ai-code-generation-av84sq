# Terraform configuration file for AWS VPC infrastructure 
# Defines a secure, multi-tier network architecture for the Document Management Platform
# AWS provider version: ~> 4.0

# Local variables for resource naming consistency and computed values
locals {
  # Common tags to be applied to all resources
  common_tags = {
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
  
  # Calculate the number of NAT gateways based on configuration
  nat_gateway_count = var.single_nat_gateway ? 1 : length(var.availability_zones)
}

# Creates the main VPC with DNS support and hostnames enabled
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-vpc"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates public subnets across multiple availability zones for load balancers and bastion hosts
resource "aws_subnet" "public" {
  count                   = length(var.availability_zones)
  vpc_id                  = aws_vpc.main.id
  cidr_block              = var.public_subnet_cidrs[count.index]
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true
  
  tags = {
    Name                     = "${var.project_name}-${var.environment}-public-${var.availability_zones[count.index]}"
    Environment              = var.environment
    Project                  = var.project_name
    "kubernetes.io/role/elb" = "1"
  }
}

# Creates private application subnets for EKS nodes and application services
resource "aws_subnet" "private_app" {
  count             = length(var.availability_zones)
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]
  
  tags = {
    Name                              = "${var.project_name}-${var.environment}-private-app-${var.availability_zones[count.index]}"
    Environment                       = var.environment
    Project                           = var.project_name
    "kubernetes.io/role/internal-elb" = "1"
  }
}

# Creates private data subnets for databases, Elasticsearch, and other data services
resource "aws_subnet" "private_data" {
  count             = length(var.availability_zones)
  vpc_id            = aws_vpc.main.id
  cidr_block        = var.private_data_subnet_cidrs[count.index]
  availability_zone = var.availability_zones[count.index]
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-private-data-${var.availability_zones[count.index]}"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates an internet gateway for public subnet internet access
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-igw"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates elastic IPs for NAT gateways
resource "aws_eip" "nat" {
  count = var.single_nat_gateway ? 1 : length(var.availability_zones)
  vpc   = true
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-nat-eip-${count.index}"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates NAT gateways for private subnet outbound internet access
resource "aws_nat_gateway" "nat" {
  count         = var.single_nat_gateway ? 1 : length(var.availability_zones)
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-nat-${count.index}"
    Environment = var.environment
    Project     = var.project_name
  }
  
  # Ensure the Internet Gateway is created first
  depends_on = [aws_internet_gateway.main]
}

# Creates a route table for public subnets with internet gateway route
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-public-rt"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Adds a route to the internet gateway in the public route table
resource "aws_route" "public_internet_gateway" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

# Creates route tables for private application subnets with NAT gateway routes
resource "aws_route_table" "private_app" {
  count  = var.single_nat_gateway ? 1 : length(var.availability_zones)
  vpc_id = aws_vpc.main.id
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-private-app-rt-${count.index}"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Adds routes to NAT gateways in the private application route tables
resource "aws_route" "private_app_nat_gateway" {
  count                  = var.single_nat_gateway ? 1 : length(var.availability_zones)
  route_table_id         = aws_route_table.private_app[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = var.single_nat_gateway ? aws_nat_gateway.nat[0].id : aws_nat_gateway.nat[count.index].id
}

# Creates route tables for private data subnets with NAT gateway routes
resource "aws_route_table" "private_data" {
  count  = var.single_nat_gateway ? 1 : length(var.availability_zones)
  vpc_id = aws_vpc.main.id
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-private-data-rt-${count.index}"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Adds routes to NAT gateways in the private data route tables
resource "aws_route" "private_data_nat_gateway" {
  count                  = var.single_nat_gateway ? 1 : length(var.availability_zones)
  route_table_id         = aws_route_table.private_data[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = var.single_nat_gateway ? aws_nat_gateway.nat[0].id : aws_nat_gateway.nat[count.index].id
}

# Associates public subnets with the public route table
resource "aws_route_table_association" "public" {
  count          = length(var.availability_zones)
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Associates private application subnets with their route tables
resource "aws_route_table_association" "private_app" {
  count          = length(var.availability_zones)
  subnet_id      = aws_subnet.private_app[count.index].id
  route_table_id = var.single_nat_gateway ? aws_route_table.private_app[0].id : aws_route_table.private_app[count.index].id
}

# Associates private data subnets with their route tables
resource "aws_route_table_association" "private_data" {
  count          = length(var.availability_zones)
  subnet_id      = aws_subnet.private_data[count.index].id
  route_table_id = var.single_nat_gateway ? aws_route_table.private_data[0].id : aws_route_table.private_data[count.index].id
}

# Creates a VPC endpoint for secure access to S3 without internet
resource "aws_vpc_endpoint" "s3" {
  count             = var.enable_s3_endpoint ? 1 : 0
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = concat(
    aws_route_table.private_app[*].id,
    aws_route_table.private_data[*].id
  )
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-s3-endpoint"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates VPC flow logs for network traffic monitoring and security compliance
resource "aws_flow_log" "vpc_flow_logs" {
  count                = var.enable_flow_logs ? 1 : 0
  log_destination_type = "cloud-watch-logs"
  log_destination      = aws_cloudwatch_log_group.flow_logs[0].arn
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.main.id
  iam_role_arn         = aws_iam_role.flow_logs[0].arn
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-flow-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates a CloudWatch log group for VPC flow logs
resource "aws_cloudwatch_log_group" "flow_logs" {
  count             = var.enable_flow_logs ? 1 : 0
  name              = "/aws/vpc-flow-logs/${var.project_name}-${var.environment}"
  retention_in_days = var.flow_logs_retention_days
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-flow-logs-group"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates an IAM role for VPC flow logs
resource "aws_iam_role" "flow_logs" {
  count = var.enable_flow_logs ? 1 : 0
  name  = "${var.project_name}-${var.environment}-flow-logs-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-flow-logs-role"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Creates an IAM policy for the VPC flow logs role
resource "aws_iam_role_policy" "flow_logs" {
  count = var.enable_flow_logs ? 1 : 0
  name  = "${var.project_name}-${var.environment}-flow-logs-policy"
  role  = aws_iam_role.flow_logs[0].id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Effect   = "Allow"
        Resource = "*"
      }
    ]
  })
}

# Creates a default security group for the VPC
resource "aws_security_group" "default" {
  name        = "${var.project_name}-${var.environment}-default-sg"
  description = "Default security group for ${var.project_name}-${var.environment} VPC"
  vpc_id      = aws_vpc.main.id
  
  # By default, no inbound traffic is allowed
  
  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-default-sg"
    Environment = var.environment
    Project     = var.project_name
  }
}