#!/bin/bash
# Verify cross-cloud connectivity for Raft cluster

INVENTORY_FILE="inventory.ini"
SSH_KEY="~/.ssh/${KEY_NAME}.pem"

echo "=== Cross-Cloud Connectivity Verification ==="
echo ""

# Check if inventory file exists
if [ ! -f "$INVENTORY_FILE" ]; then
    echo "ERROR: inventory.ini not found. Please ensure Ansible inventory is generated."
    echo "Run: terraform output to see instance IPs and create inventory.ini"
    exit 1
fi

# Extract IPs from inventory
AWS_IPS=$(awk '/^\[aws\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)
GCP_IPS=$(awk '/^\[gcp\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)

ALL_IPS="$AWS_IPS $GCP_IPS"

echo "Testing connectivity from each node to all others..."
echo ""

for source_ip in $ALL_IPS; do
    # Determine SSH user based on IP
    if echo "$AWS_IPS" | grep -q "$source_ip"; then
        SSH_USER="ec2-user"
    else
        SSH_USER="${GCP_USER_NAME:-yourusername}"
    fi

    echo "--- Testing from $source_ip ($SSH_USER) ---"

    for target_ip in $ALL_IPS; do
        if [ "$source_ip" = "$target_ip" ]; then
            continue
        fi

        # Test ping (ICMP)
        ssh -i $SSH_KEY -o StrictHostKeyChecking=no $SSH_USER@$source_ip \
            "ping -c 3 -W 2 $target_ip > /dev/null 2>&1 && echo '✅ PING $target_ip' || echo '❌ PING $target_ip'"

        # Test TCP connection to app port (8080)
        ssh -i $SSH_KEY -o StrictHostKeyChecking=no $SSH_USER@$source_ip \
            "timeout 5 bash -c 'cat < /dev/null > /dev/tcp/$target_ip/8080' 2>/dev/null && echo '✅ TCP $target_ip:8080' || echo '⚠️  TCP $target_ip:8080 (no listener yet)'"
    done
    echo ""
done

echo "=== Connectivity Test Complete ==="
