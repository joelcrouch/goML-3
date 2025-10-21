terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  required_version = ">= 1.6.0"
}

# --- Providers Configuration ---

provider "aws" {
  region = var.aws_region
}

provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone    = var.gcp_zone
  # Assumes GCP_APPLICATION_CREDENTIALS or similar env var is set,
  # or you can use the optional 'credentials' argument with a local path.
}

# -----------------------------------------------------------------------------
# AWS Resources (3 Nodes)
# -----------------------------------------------------------------------------

# --- AMI Lookup (Amazon Linux ARM) ---
data "aws_ami" "amazon_linux_arm" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-arm64-gp2"]
  }
}

# --- Try to find existing Security Group ---
#data "aws_security_group" "existing" {
 # filter {
 #   name   = "group-name"
 #   values = [var.aws_security_group_name]
 # }
#}

# --- Create Security Group only if missing ---
resource "aws_security_group" "multi_cloud_sg_aws" {
  name = var.aws_security_group_name
  description   = "Security group for multi-cloud experiment (AWS)"

  # SSH, App Port, and ICMP Ingress rules
  ingress { 
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"] 
  }
  ingress { 
    from_port = var.app_port
    to_port = var.app_port
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress { 
    from_port = -1
    to_port = -1
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"] 
  }

  # All Outbound traffic allowed
  egress { 
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"] 
  }

  tags = {
    Name    = var.aws_security_group_name
    Project = "MultiCloudDistSys"
    Cloud   = "AWS"
  }
}

# --- Select SG dynamically ---
locals {
  aws_sg_id = aws_security_group.multi_cloud_sg_aws.id
}
#locals {
 # aws_sg_id = length(try(data.aws_security_group.existing.id, "")) > 0 ?
 #   data.aws_security_group.existing.id :
 #   aws_security_group.multi_cloud_sg_aws[0].id
#}
#l#ocals 
  #aws_sg_id = length(try(data.aws_security_group.existing.id,
#"")) > 0 ? data.aws_security_group.existing.id : aws_security_group.multi_cloud_sg_aws[0].id



# --- AWS User Data Script ---
locals {
  aws_user_data = <<-EOF
    #!/bin/bash
    yum update -y
    yum install -y git python3-pip
    # Note: Go installation should be handled by Ansible later
    git clone ${var.project_repo} /home/ec2-user/multiCloudDistSys
    pip3 install -r /home/ec2-user/multiCloudDistSys/requirements.txt || echo "No requirements.txt found or pip failed."
    echo "Setup complete on AWS instance $(hostname)"
  EOF
}

# --- Launch EC2 Instances ---
resource "aws_instance" "multi_cloud_node_aws" {
  ami                         = data.aws_ami.amazon_linux_arm.id
  instance_type               = var.aws_instance_type
  key_name                    = var.key_name
  count                       = var.aws_instance_count
  vpc_security_group_ids      = [local.aws_sg_id]
  user_data                   = local.aws_user_data
  associate_public_ip_address = true

  tags = {
    Project = "MultiCloudDistSys"
    Name    = "MultiCloudNode-AWS-${count.index + 1}"
    Tier    = "raft"
  }
}

# -----------------------------------------------------------------------------
# GCP Resources (2 Nodes)
# -----------------------------------------------------------------------------

# --- Firewall (using the unique name) ---
resource "google_compute_firewall" "multi_cloud_gcp" {
  name    = var.gcp_firewall_name
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["22", tostring(var.app_port)]
  }

  # Egress traffic is allowed by default on the default network
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["multi-cloud-node-gcp"]

  description = "Allow SSH and app traffic for multi-cloud experiment (GCP)"
}

# --- GCP Startup Script ---
locals {
  gcp_startup_script = <<-EOF
    #!/bin/bash
    apt-get update -y
    apt-get install -y git python3 python3-pip
    # Note: Go installation should be handled by Ansible later
    git clone ${var.project_repo} /home/${var.gcp_user_name}/multiCloudDistSys
    pip3 install -r /home/${var.gcp_user_name}/multiCloudDistSys/requirements.txt || echo "No requirements.txt found or pip failed."
    echo "Setup complete on GCP instance $(hostname)"
  EOF
}

# --- Launch GCP Instances ---
resource "google_compute_instance" "multi_cloud_node_gcp" {
  count        = var.gcp_instance_count
  name         = "multi-cloud-node-gcp-${count.index + 1}"
  machine_type = var.gcp_instance_type
  zone         = var.gcp_zone

  tags = ["multi-cloud-node-gcp"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  network_interface {
    network       = "default"
    access_config {} # Assign public IP
  }

  metadata = {
    startup-script = local.gcp_startup_script
    ssh-keys       = "${var.gcp_user_name}:${file("~/.ssh/${var.key_name}.pub")}"
  }

  labels = {
    project = "multi-cloud-distsys"
    tier    = "raft"
  }
}

# -----------------------------------------------------------------------------
# Ansible Provisioning (Local Execution)
# -----------------------------------------------------------------------------

resource "null_resource" "run_ansible_provisioning" {
  # Add a trigger that changes when any of the instance IPs change,
  # forcing Ansible to re-run if the instances are rebuilt.
  triggers = {
    aws_ips = join(",", aws_instance.multi_cloud_node_aws[*].public_ip)
    gcp_ips = join(",", google_compute_instance.multi_cloud_node_gcp[*].network_interface[0].access_config[0].nat_ip)
  }
provisioner "local-exec" {
  command = <<-EOT
    echo "--- Generating Ansible Inventory (inventory.ini) ---"

    cat > inventory.ini <<-INVENTORY
    [aws]
    ${join("\n", aws_instance.multi_cloud_node_aws[*].public_ip)}

    [gcp]
    ${join("\n", google_compute_instance.multi_cloud_node_gcp[*].network_interface[0].access_config[0].nat_ip)}

    [aws:vars]
    ansible_user = ec2-user

    [gcp:vars]
    ansible_user = ${var.gcp_user_name}

    [all:vars]
    ansible_ssh_private_key_file = ~/.ssh/${var.key_name}.pem
    INVENTORY

    echo "--- Running Ansible Playbook ---"
    # Set a remote temp directory for Ansible and run the playbook
    ANSIBLE_REMOTE_TMP=/tmp ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook \
      -i inventory.ini ../ansible/setup_raft_storage.yml \
      --extra-vars "app_repo=${var.project_repo}"
  EOT
}

  depends_on = [
    aws_instance.multi_cloud_node_aws,
    google_compute_instance.multi_cloud_node_gcp
  ]
}