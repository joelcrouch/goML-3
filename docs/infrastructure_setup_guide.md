# Multi-Cloud Infrastructure Setup Guide
## Story 1.1: Infrastructure Provisioning and Verification

**Status**: Scripts created, ready for execution
**Last Updated**: 2025-10-19
**Prerequisites**: Terraform infrastructure deployed (5 nodes: 3 AWS, 2 GCP)

---

## Overview

This guide walks through completing Story 1.1 of Sprint 1, which focuses on verifying and configuring the multi-cloud infrastructure for Raft cluster deployment.

## Prerequisites Checklist

Before running these scripts, ensure:

- [ ] Terraform infrastructure deployed successfully
- [ ] 3 AWS EC2 instances (t4g.small, ARM64) running
- [ ] 2 GCP Compute instances (e2-medium, AMD64) running
- [ ] SSH key pair configured (`~/.ssh/${KEY_NAME}.pem`)
- [ ] `inventory.ini` file generated with all instance IPs
- [ ] Environment variables set:
  - `KEY_NAME`: Your SSH key name
  - `GCP_USER_NAME`: Your GCP Linux username

### Generate inventory.ini

If you don't have an inventory.ini file yet, create it manually or run:

```bash
cd terraform

# Get AWS IPs
terraform output aws_instance_public_ips

# Get GCP IPs
terraform output gcp_instance_public_ips
```

Create `inventory.ini` at the project root:

```ini
[aws]
<AWS_IP_1>
<AWS_IP_2>
<AWS_IP_3>

[gcp]
<GCP_IP_1>
<GCP_IP_2>

[all:vars]
ansible_user=ec2-user
ansible_ssh_private_key_file=~/.ssh/multi-cloud-key.pem
```

---

## Step-by-Step Execution

### Step 1: Verify Cross-Cloud Connectivity

**Purpose**: Ensure all nodes can reach each other via ICMP and TCP.

**Script**: `scripts/verify_connectivity.sh`

**What it does**:
- Tests ping (ICMP) between all node pairs
- Tests TCP connectivity on port 8080 (Raft port)
- Reports success/failure for each connection

**Run**:
```bash
export KEY_NAME="multi-cloud-key"
export GCP_USER_NAME="yourusername"  # Replace with your GCP username

./scripts/verify_connectivity.sh
```

**Expected Output**:
```
=== Cross-Cloud Connectivity Verification ===

--- Testing from 3.85.123.45 (ec2-user) ---
✅ PING 3.85.123.46
⚠️  TCP 3.85.123.46:8080 (no listener yet)
✅ PING 34.122.34.56
⚠️  TCP 34.122.34.56:8080 (no listener yet)
...
```

**Success Criteria**:
- ✅ All PING tests succeed
- ⚠️ TCP tests show "no listener" (expected - Raft not running yet)
- ❌ indicates a problem - check security groups/firewalls

**Troubleshooting**:
- **Ping fails**: Check security groups allow ICMP
- **Timeout**: Verify IPs are correct in inventory.ini
- **Permission denied**: Ensure SSH key has correct permissions (`chmod 400`)

---

### Step 2: Setup Raft Persistent Storage

**Purpose**: Create directories for Raft logs, snapshots, and state database.

**Script**: `ansible/setup_raft_storage.yml`

**What it does**:
- Creates `/var/lib/raft/` with subdirectories:
  - `logs/` - Raft log entries
  - `snapshots/` - Periodic state snapshots
  - `boltdb/` - BoltDB persistent storage
- Creates `/etc/raft/` for configuration files
- Sets correct ownership and permissions

**Run**:
```bash
ansible-playbook -i inventory.ini ansible/setup_raft_storage.yml
```

**Expected Output**:
```
PLAY [Setup Raft Persistent Storage] ********************

TASK [Create Raft data directories] ********************
changed: [3.85.123.45] => (item=/var/lib/raft)
changed: [3.85.123.45] => (item=/var/lib/raft/logs)
...

TASK [Display storage structure] ************************
ok: [3.85.123.45] => {
    "msg": "Raft storage setup complete: ..."
}

PLAY RECAP **********************************************
3.85.123.45  : ok=4  changed=2  unreachable=0  failed=0
...
```

**Success Criteria**:
- ✅ All directories created successfully
- ✅ Correct ownership (`ec2-user` or GCP username)
- ✅ No permission errors

**Verify manually** (optional):
```bash
ssh -i ~/.ssh/${KEY_NAME}.pem ec2-user@<AWS_IP> "ls -la /var/lib/raft/"
```

---

### Step 3: Generate Node Configurations

**Purpose**: Create node-specific JSON configs with peer information.

**Script**: `scripts/generate_node_configs.py`

**What it does**:
- Reads Terraform outputs to get all node IPs
- Generates `config/nodes/*.json` for each node
- Creates deployment script `scripts/deploy_configs.sh`

**Configuration includes**:
- Node ID (e.g., `aws-node-1`, `gcp-node-2`)
- Bind and advertise addresses
- Peer list (other nodes in cluster)
- Raft timing parameters (heartbeat, election timeout)
- gRPC port configuration

**Run**:
```bash
python3 scripts/generate_node_configs.py
```

**Expected Output**:
```
✅ Generated config/nodes/aws-node-1.json
✅ Generated config/nodes/aws-node-2.json
✅ Generated config/nodes/aws-node-3.json
✅ Generated config/nodes/gcp-node-1.json
✅ Generated config/nodes/gcp-node-2.json

✅ Generated deployment script: scripts/deploy_configs.sh

Run: ./scripts/deploy_configs.sh
```

**Check generated configs**:
```bash
cat config/nodes/aws-node-1.json
```

**Example config**:
```json
{
  "node_id": "aws-node-1",
  "bind_address": "3.85.123.45:8080",
  "advertise_address": "3.85.123.45:8080",
  "cloud_provider": "aws",
  "region": "us-east-1",
  "data_dir": "/var/lib/raft",
  "bootstrap_expect": 5,
  "peers": [
    "3.85.123.46:8080",
    "3.85.123.47:8080",
    "34.122.34.56:8080",
    "34.122.34.57:8080"
  ],
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
```

---

### Step 4: Deploy Node Configurations

**Purpose**: Copy generated configs to each node.

**Script**: `scripts/deploy_configs.sh` (auto-generated)

**What it does**:
- SCPs each config to its corresponding node
- Moves config to `/etc/raft/config.json`
- Sets correct permissions

**Run**:
```bash
./scripts/deploy_configs.sh
```

**Expected Output**:
```
Deploying config to aws-node-1 (3.85.123.45)...
✅ aws-node-1 configured

Deploying config to aws-node-2 (3.85.123.46)...
✅ aws-node-2 configured

...

=== All nodes configured ===
```

**Verify deployment**:
```bash
ssh -i ~/.ssh/${KEY_NAME}.pem ec2-user@<AWS_IP> "cat /etc/raft/config.json"
```

---

### Step 5: Measure Baseline Network Latency

**Purpose**: Document network performance between all node pairs.

**Script**: `scripts/measure_latency.sh`

**What it does**:
- Measures round-trip ping latency (10 samples per pair)
- Creates latency matrix showing all node-to-node latencies
- Saves results to `docs/baseline_latency.md`

**Run**:
```bash
./scripts/measure_latency.sh
```

**Expected Output**:
```
✅ Latency measurements saved to docs/baseline_latency.md

# Baseline Network Latency Measurements

**Measured**: Sat Oct 19 20:45:00 UTC 2025

## Latency Matrix (ms)

| From \ To | aws-1 | aws-2 | aws-3 | gcp-1 | gcp-2 |
|-----------|-------|-------|-------|-------|-------|
| **aws-1** | - | 2.1 | 2.3 | 65.4 | 66.1 |
| **aws-2** | 2.2 | - | 2.0 | 64.8 | 65.9 |
| **aws-3** | 2.1 | 2.2 | - | 65.2 | 66.3 |
| **gcp-1** | 65.5 | 65.0 | 65.4 | - | 1.8 |
| **gcp-2** | 66.0 | 66.2 | 66.1 | 1.9 | - |

## Analysis

- **Intra-cloud (AWS-AWS)**: Expected <5ms ✅
- **Intra-cloud (GCP-GCP)**: Expected <5ms ✅
- **Cross-cloud (AWS-GCP)**: Expected 50-100ms ✅
```

**What good latencies look like**:
- AWS to AWS: 1-5ms (same region)
- GCP to GCP: 1-5ms (same region)
- AWS to GCP: 50-100ms (cross-cloud, acceptable)

**Performance implications**:
- Raft heartbeat: 1s interval (sufficient for 100ms latency)
- Leader election: 3s timeout (allows for network variance)
- Commit latency: ~500ms + network latency

---

## Validation Checklist

Before proceeding to Story 1.2 (Raft implementation), verify:

### Infrastructure
- [ ] 5 nodes deployed (3 AWS t4g.small, 2 GCP e2-medium)
- [ ] All nodes accessible via SSH
- [ ] Security groups allow traffic on ports 22, 8080, 50051

### Connectivity
- [ ] All nodes can ping each other (ICMP)
- [ ] Latency measurements documented
- [ ] Intra-cloud latency <5ms
- [ ] Cross-cloud latency <100ms

### Storage
- [ ] `/var/lib/raft/` directories created on all nodes
- [ ] `/etc/raft/` directories created on all nodes
- [ ] Correct ownership and permissions

### Configuration
- [ ] 5 node config files generated in `config/nodes/`
- [ ] All configs deployed to `/etc/raft/config.json`
- [ ] Each node has correct peer list
- [ ] Bootstrap expect = 5 for all nodes

### Documentation
- [ ] `docs/baseline_latency.md` generated
- [ ] All scripts executable and tested
- [ ] inventory.ini accurate

---

## Troubleshooting

### Connection Refused
**Problem**: SSH connection refused or timeout
**Solution**:
- Verify security group allows SSH (port 22) from your IP
- Check instance is running: `terraform output`
- Verify SSH key path: `ls -la ~/.ssh/${KEY_NAME}.pem`

### Permission Denied (Ansible)
**Problem**: Ansible can't create directories
**Solution**:
- Ensure `become: yes` in playbook (already included)
- Check SSH user has sudo privileges
- AWS: `ec2-user` has sudo by default
- GCP: Add your user to sudoers if needed

### Terraform Output Error
**Problem**: `generate_node_configs.py` can't read Terraform outputs
**Solution**:
- Run from project root (not terraform/ directory)
- Ensure Terraform state exists: `terraform state list`
- Re-run: `cd terraform && terraform apply`

### High Cross-Cloud Latency
**Problem**: AWS-GCP latency >200ms
**Impact**: Raft consensus will be slower but still functional
**Solution**:
- Consider using regions closer together
- us-east-1 (AWS) + us-central1 (GCP) = optimal
- Europe: eu-west-1 (AWS) + europe-west1 (GCP)

### Node Config Mismatch
**Problem**: Different peer counts in node configs
**Solution**:
- Delete `config/nodes/` directory
- Re-run: `python3 scripts/generate_node_configs.py`
- Re-deploy: `./scripts/deploy_configs.sh`

---

## Next Steps

With Story 1.1 complete, you're ready for **Story 1.2: Raft Cluster Implementation**.

See `docs/sprint1_raft_detailed.md` for Story 1.2 details:
1. Implement Go Raft cluster using HashiCorp Raft library
2. Leader election and state transitions
3. Log persistence with BoltDB
4. Network transport via gRPC

**Development workflow**:
```bash
# Create feature branch
git checkout -b feature/raft-control-plane

# Start implementing Story 1.2
# See docs/sprint1_raft_detailed.md for tasks
```

---

## Files Created

This guide references the following files:

```
goML#3/
├── scripts/
│   ├── verify_connectivity.sh      # Step 1: Connectivity tests
│   ├── generate_node_configs.py    # Step 3: Config generation
│   ├── deploy_configs.sh           # Step 4: Config deployment (generated)
│   └── measure_latency.sh          # Step 5: Latency measurement
├── ansible/
│   └── setup_raft_storage.yml      # Step 2: Storage setup
├── config/
│   └── nodes/                      # Generated node configs
│       ├── aws-node-1.json
│       ├── aws-node-2.json
│       ├── aws-node-3.json
│       ├── gcp-node-1.json
│       └── gcp-node-2.json
├── docs/
│   ├── infrastructure_setup_guide.md  # This file
│   └── baseline_latency.md            # Generated latency report
└── inventory.ini                      # Ansible inventory (manual)
```

---

## Summary

**Story 1.1 Deliverables**:
1. ✅ Cross-cloud connectivity verified
2. ✅ Raft persistent storage configured
3. ✅ Node configurations generated and deployed
4. ✅ Network latency baseline documented

**Time to Complete**: ~30 minutes (assuming infrastructure already deployed)

**Infrastructure Status**: Ready for Raft cluster deployment (Story 1.2)

**Performance Baseline**:
- 5 nodes operational across 2 clouds
- Intra-cloud latency: <5ms
- Cross-cloud latency: 50-100ms
- All nodes reachable and configured

---

**Last Updated**: 2025-10-19
**Next**: Story 1.2 - Raft Cluster Implementation
