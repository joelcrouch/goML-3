# Multi-Cloud ML Orchestration with Raft Consensus

**Architecture**: Go Control Plane + Python Data Plane
**Current Sprint**: Sprint 1 - Raft-Based Control Plane
**Current Story**: 1.1 - Multi-Cloud Infrastructure Setup

---

## Quick Start

### Prerequisites
- Terraform 1.5+ installed
- Ansible 2.15+ installed
- Python 3.11+ installed
- Go 1.21+ installed
- AWS and GCP credentials configured

### Infrastructure Status
✅ **Story 1.1 Scripts Created** (2025-10-19)

The following scripts are ready to execute:

```bash
# 1. Verify cross-cloud connectivity
./scripts/verify_connectivity.sh

# 2. Setup Raft storage directories
ansible-playbook -i inventory.ini ansible/setup_raft_storage.yml

# 3. Generate node configurations
python3 scripts/generate_node_configs.py

# 4. Deploy configurations to nodes
./scripts/deploy_configs.sh

# 5. Measure baseline network latency
./scripts/measure_latency.sh
```

### Full Setup Guide
See **[docs/infrastructure_setup_guide.md](docs/infrastructure_setup_guide.md)** for complete step-by-step instructions.

---

## Project Structure

```
goML#3/
├── scripts/                    # Automation scripts
│   ├── verify_connectivity.sh      # Cross-cloud connectivity tests
│   ├── generate_node_configs.py    # Node config generator
│   └── measure_latency.sh          # Latency measurement
├── ansible/                    # Ansible playbooks
│   └── setup_raft_storage.yml      # Raft storage setup
├── terraform/                  # Infrastructure as Code
│   ├── main.tf                     # Multi-cloud VM provisioning
│   ├── variables.tf                # Configuration variables
│   └── output.tf                   # Terraform outputs
├── config/                     # Generated configurations
│   └── nodes/                      # Node-specific configs (generated)
├── docs/                       # Documentation
│   ├── living_doc_updated.md       # Project state and context
│   ├── sprint1_raft_detailed.md    # Sprint 1 implementation plan
│   ├── infrastructure_setup_guide.md # Setup instructions
│   └── story_1.1_completion_checklist.md # Completion tracking
└── README.md                   # This file
```

---

## Architecture Overview

### Control Plane (Go)
- **Raft consensus cluster** for fault-tolerant state management
- **Leader election** with automatic failover (<5s)
- **Global Task Manifest** replicated across all nodes
- **Cross-cloud coordination** via gRPC

### Node Agents (Python)
- **Data processing workers** with ML/data science libraries
- **Heartbeat to Control Plane** for health monitoring
- **Execute tasks** from the Global Task Manifest
- **Report results** back to Raft Leader

### Infrastructure (5 Nodes)
- **3 AWS EC2 instances** (t4g.small, ARM64, us-east-1)
- **2 GCP Compute instances** (e2-medium, AMD64, us-central1)
- **Cross-cloud networking** with latency <100ms
- **Persistent storage** for Raft logs and snapshots

---

## Sprint 1 Progress

### Story 1.1: Multi-Cloud Infrastructure (8 points)
**Status**: Scripts created, ready for execution

#### Deliverables Created
- ✅ Cross-cloud connectivity verification script
- ✅ Ansible playbook for Raft storage setup
- ✅ Node configuration generator
- ✅ Network latency measurement script
- ✅ Complete setup documentation

#### Pending Execution
- [ ] Create inventory.ini with instance IPs
- [ ] Run connectivity verification
- [ ] Setup Raft storage on all nodes
- [ ] Generate and deploy node configs
- [ ] Measure baseline latency

See **[docs/story_1.1_completion_checklist.md](docs/story_1.1_completion_checklist.md)** for detailed tracking.

---

### Story 1.2: Raft Cluster Implementation (8 points)
**Status**: Next up

**Plan**:
- HashiCorp Raft library integration
- Leader election and state transitions
- Log persistence (BoltDB backend)
- Network transport (TCP via gRPC)

See **[docs/sprint1_raft_detailed.md](docs/sprint1_raft_detailed.md)** for details.

---

## Key Documentation

### Essential Reading
1. **[docs/living_doc_updated.md](docs/living_doc_updated.md)** - Project state, architecture pivot, full context
2. **[docs/infrastructure_setup_guide.md](docs/infrastructure_setup_guide.md)** - Step-by-step setup instructions
3. **[docs/sprint1_raft_detailed.md](docs/sprint1_raft_detailed.md)** - Sprint 1 implementation plan

### Architecture & Design
- **[docs/architecture.md](docs/architecture.md)** - System architecture
- **[docs/researchReport.md](docs/researchReport.md)** - Research and rationale
- **[docs/whycahngethearchitecture.md](docs/whycahngethearchitecture.md)** - Architecture pivot explanation

---

## Quick Commands Reference

### Terraform Operations
```bash
cd terraform

# Deploy infrastructure
terraform init
terraform plan -out=plan.out
terraform apply plan.out

# Get instance IPs
terraform output aws_instance_public_ips
terraform output gcp_instance_public_ips

# Destroy infrastructure (when done)
terraform destroy
```

### Ansible Operations
```bash
# Setup Raft storage
ansible-playbook -i inventory.ini ansible/setup_raft_storage.yml

# Check connectivity to all nodes
ansible all -i inventory.ini -m ping
```

### Node Configuration
```bash
# Generate configs
python3 scripts/generate_node_configs.py

# Deploy configs
./scripts/deploy_configs.sh

# Verify config on a node
ssh -i ~/.ssh/${KEY_NAME}.pem ec2-user@<IP> "cat /etc/raft/config.json"
```

---

## Environment Variables

Set these before running scripts:

```bash
export KEY_NAME="multi-cloud-key"           # Your SSH key name
export GCP_USER_NAME="yourusername"         # Your GCP Linux username
export AWS_ACCESS_KEY_ID="..."              # AWS credentials
export AWS_SECRET_ACCESS_KEY="..."          # AWS credentials
export GOOGLE_APPLICATION_CREDENTIALS="..." # GCP credentials path
```

---

## Performance Targets

### Story 1.1 Targets
| Metric | Target |
|--------|--------|
| Intra-cloud latency | <5ms |
| Cross-cloud latency | <100ms |
| All nodes reachable | 100% |

### Story 1.2 Targets (Upcoming)
| Metric | Target |
|--------|--------|
| Leader election time | <10s |
| Leader failover time | <5s |
| Task submission latency | <500ms |
| Heartbeat overhead | <1% CPU |

---

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Control Plane | Go 1.21+ |
| Consensus | HashiCorp Raft |
| Storage | BoltDB |
| Node Agents | Python 3.11+ |
| Computation | NumPy, SciPy |
| Communication | gRPC |
| Infrastructure | Terraform |
| Configuration | Ansible |
| Cloud Providers | AWS EC2, GCP Compute Engine |

---

## Contributing

### Development Workflow
1. Read documentation (start with `docs/living_doc_updated.md`)
2. Create feature branch (`feature/story-name`)
3. Implement story tasks
4. Write tests (target 75% coverage)
5. Update documentation
6. Submit for review

### Commit Strategy
- Incremental commits per story task
- All commits must pass tests
- Update docs with each major change
- Merge to main after story complete

---

## Support & Resources

### Project Resources
- **Living Document**: [docs/living_doc_updated.md](docs/living_doc_updated.md)
- **Sprint 1 Plan**: [docs/sprint1_raft_detailed.md](docs/sprint1_raft_detailed.md)
- **Setup Guide**: [docs/infrastructure_setup_guide.md](docs/infrastructure_setup_guide.md)

### External Resources
- **Raft Paper**: https://raft.github.io/raft.pdf
- **HashiCorp Raft**: https://github.com/hashicorp/raft
- **gRPC Documentation**: https://grpc.io/
- **NumPy Documentation**: https://numpy.org/doc/

---

## Current Status Summary

**Infrastructure**: Deployed (5 nodes: 3 AWS, 2 GCP)
**Story 1.1**: Scripts created, ready for execution
**Story 1.2**: Next up - Raft implementation
**Sprint 1**: In progress (2 of 4 stories planned)

**Last Updated**: 2025-10-19

---

## License

[To be determined]

---

## Contact

**Project Repository**: [To be added]
**Documentation**: See `docs/` directory
**Issues**: Track in issue tracker (TBD)

---

**Next Action**: Create `inventory.ini` and run Story 1.1 scripts (see [infrastructure setup guide](docs/infrastructure_setup_guide.md))
