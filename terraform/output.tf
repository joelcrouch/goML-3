output "aws_instance_public_ips" {
  description = "Public IPs of the deployed AWS EC2 instances"
  value       = [for i in aws_instance.multi_cloud_node_aws : i.public_ip]
}

output "gcp_instance_public_ips" {
  description = "Public IPs of the deployed GCP Compute Instances"
  # GCP's public IP is in the access_config block of the network_interface
  value       = [for i in google_compute_instance.multi_cloud_node_gcp : i.network_interface[0].access_config[0].nat_ip]
}

output "all_public_ips" {
  description = "A combined, ordered list of all 6 public IPs for use in Ansible/Raft configuration"
  value       = concat(
    [for i in aws_instance.multi_cloud_node_aws : i.public_ip],
    [for i in google_compute_instance.multi_cloud_node_gcp : i.network_interface[0].access_config[0].nat_ip]
  )
}