# -----------------------------------------------------------------------------
# Global Variables
# -----------------------------------------------------------------------------

variable "key_name" {
  description = "SSH key pair name (used for both AWS and GCP metadata)"
  type        = string
  default     = "multi-cloud-key" # CHANGE ME
}

variable "project_repo" {
  description = "GitHub repository for your project (used for initial git clone)"
  type        = string
  default     = "https://github.com/joelcrouch/goML-3"
}

variable "app_port" {
  description = "Application port for Raft/API communication (must be open in both firewalls)"
  type        = number
  default     = 8080
}

# -----------------------------------------------------------------------------
# AWS Variables (3 Nodes)
# -----------------------------------------------------------------------------

variable "aws_region" {
  description = "AWS region to deploy resources in"
  type        = string
  default     = "us-east-1"
}

variable "aws_instance_type" {
  description = "EC2 instance type (t4g.nano is ARM and Free Tier eligible)"
  type        = string
  default     = "t4g.nano"
}

variable "aws_instance_count" {
  description = "Number of EC2 instances to launch"
  type        = number
  default     = 3 # Kept at 3
}

variable "aws_security_group_name" {
  description = "Name of the security group to use (AWS)"
  type        = string
  default     = "multi-cloud-sg-aws"
}

# -----------------------------------------------------------------------------
# GCP Variables (3 Nodes)
# -----------------------------------------------------------------------------

variable "gcp_project_id" {
  description = "The ID of the GCP project"
  type        = string
  default     = "norse-ward-472620-r0" # CRITICAL: REPLACE WITH YOUR ACTUAL PROJECT ID
}

variable "gcp_region" {
  description = "GCP region to deploy resources in"
  type        = string
  default     = "us-central1"
}

variable "gcp_zone" {
  description = "GCP zone to deploy instances in"
  type        = string
  default     = "us-central1-a"
}

variable "gcp_instance_type" {
  description = "GCP Compute Engine machine type (e2-micro is economical)"
  type        = string
  default     = "e2-micro"
}

variable "gcp_instance_count" {
  description = "Number of GCP instances to launch"
  type        = number
  default     = 3 # CHANGED FROM 2 TO 3
}

variable "gcp_firewall_name" {
  description = "Name of the GCP firewall rule"
  type        = string
  default     = "multi-cloud-firewall-gcp"
}

variable "gcp_user_name" {
  description = "The primary Linux user name on the GCP instance (e.g., your OS login name, for SSH and git clone)"
  type        = string
  default     = "joelcrouch" # CRITICAL: REPLACE WITH YOUR ACTUAL LINUX USERNAME
}