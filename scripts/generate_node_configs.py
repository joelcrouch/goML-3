#!/usr/bin/env python3
"""
Generate Raft node configuration files for each instance.
Reads Terraform outputs and creates node-specific configs.
"""

import json
import subprocess
import sys
from pathlib import Path

def get_terraform_outputs():
    """Read Terraform outputs"""
    result = subprocess.run(
        ["terraform", "output", "-json"],
        capture_output=True,
        text=True,
        cwd="terraform"
    )

    if result.returncode != 0:
        print(f"Error reading Terraform outputs: {result.stderr}")
        sys.exit(1)

    return json.loads(result.stdout)

def generate_node_config(node_id, ip_address, cloud_provider, all_peer_ips, app_port=8080):
    """Generate configuration for a single node"""

    # Remove self from peers
    peers = [f"{peer_ip}:{app_port}" for peer_ip in all_peer_ips if peer_ip != ip_address]

    config = {
        "node_id": node_id,
        "bind_address": f"{ip_address}:{app_port}",
        "advertise_address": f"{ip_address}:{app_port}",
        "cloud_provider": cloud_provider,
        "region": "us-east-1" if cloud_provider == "aws" else "us-central1",
        "data_dir": "/var/lib/raft",
        "bootstrap_expect": 5,  # Total nodes in cluster
        "peers": peers,
        "raft": {
            "heartbeat_timeout": "1s",
            "election_timeout": "3s",
            "commit_timeout": "500ms",
            "snapshot_interval": "120s",
            "snapshot_threshold": 8192
        },
        "grpc": {
            "port": 50051,
            "max_concurrent_streams": 1000
        }
    }

    return config

def main():
    outputs = get_terraform_outputs()

    aws_ips = outputs['aws_instance_public_ips']['value']
    gcp_ips = outputs['gcp_instance_public_ips']['value']

    all_ips = aws_ips + gcp_ips

    # Create configs directory
    config_dir = Path("config/nodes")
    config_dir.mkdir(parents=True, exist_ok=True)

    configs = []

    # Generate AWS node configs
    for i, ip in enumerate(aws_ips, start=1):
        node_id = f"aws-node-{i}"
        config = generate_node_config(node_id, ip, "aws", all_ips)

        config_file = config_dir / f"{node_id}.json"
        with open(config_file, 'w') as f:
            json.dump(config, f, indent=2)

        configs.append((node_id, ip, "ec2-user", config_file))
        print(f"✅ Generated {config_file}")

    # Generate GCP node configs
    for i, ip in enumerate(gcp_ips, start=1):
        node_id = f"gcp-node-{i}"
        config = generate_node_config(node_id, ip, "gcp", all_ips)

        config_file = config_dir / f"{node_id}.json"
        with open(config_file, 'w') as f:
            json.dump(config, f, indent=2)

        configs.append((node_id, ip, "YOUR_GCP_USER", config_file))
        print(f"✅ Generated {config_file}")

    # Generate deployment script
    deploy_script = Path("scripts/deploy_configs.sh")
    with open(deploy_script, 'w') as f:
        f.write("#!/bin/bash\n")
        f.write("# Deploy node configurations to instances\n\n")
        f.write("set -e\n\n")

        for node_id, ip, ssh_user, config_file in configs:
            f.write(f"echo 'Deploying config to {node_id} ({ip})...'\n")
            f.write(f"scp -i ~/.ssh/${{KEY_NAME}}.pem -o StrictHostKeyChecking=no \\\n")
            f.write(f"    {config_file} {ssh_user}@{ip}:/tmp/node_config.json\n")
            f.write(f"ssh -i ~/.ssh/${{KEY_NAME}}.pem -o StrictHostKeyChecking=no \\\n")
            f.write(f"    {ssh_user}@{ip} 'sudo mv /tmp/node_config.json /etc/raft/config.json'\n")
            f.write(f"echo '✅ {node_id} configured'\n\n")

        f.write("echo '=== All nodes configured ==='\n")

    deploy_script.chmod(0o755)
    print(f"\n✅ Generated deployment script: {deploy_script}")
    print(f"\nRun: ./{deploy_script}")

if __name__ == "__main__":
    main()
