#!/bin/bash
# Measure baseline network latency between all nodes

INVENTORY_FILE="inventory.ini"
OUTPUT_FILE="docs/baseline_latency.md"

echo "# Baseline Network Latency Measurements" > $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "**Measured**: $(date)" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "## Latency Matrix (ms)" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE

# Check if inventory file exists
if [ ! -f "$INVENTORY_FILE" ]; then
    echo "ERROR: inventory.ini not found. Please ensure Ansible inventory is generated."
    exit 1
fi

# Extract IPs
AWS_IPS=$(awk '/^\[aws\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)
GCP_IPS=$(awk '/^\[gcp\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)

ALL_IPS=($AWS_IPS $GCP_IPS)
NODE_NAMES=("aws-1" "aws-2" "aws-3" "gcp-1" "gcp-2")

# Table header
echo -n "| From \\ To |" >> $OUTPUT_FILE
for name in "${NODE_NAMES[@]}"; do
    echo -n " $name |" >> $OUTPUT_FILE
done
echo "" >> $OUTPUT_FILE

# Table separator
echo -n "|-----------|" >> $OUTPUT_FILE
for name in "${NODE_NAMES[@]}"; do
    echo -n "---------|" >> $OUTPUT_FILE
done
echo "" >> $OUTPUT_FILE

# Measure latencies
for i in "${!ALL_IPS[@]}"; do
    source_ip="${ALL_IPS[$i]}"
    source_name="${NODE_NAMES[$i]}"

    # Determine SSH user
    if [[ $source_name == aws-* ]]; then
        SSH_USER="ec2-user"
    else
        SSH_USER="${GCP_USER_NAME:-yourusername}"
    fi

    echo -n "| **$source_name** |" >> $OUTPUT_FILE

    for target_ip in "${ALL_IPS[@]}"; do
        if [ "$source_ip" = "$target_ip" ]; then
            echo -n " - |" >> $OUTPUT_FILE
        else
            # Measure average latency over 10 pings
            latency=$(ssh -i ~/.ssh/${KEY_NAME}.pem -o StrictHostKeyChecking=no \
                $SSH_USER@$source_ip \
                "ping -c 10 -q $target_ip 2>/dev/null | grep 'rtt' | cut -d'/' -f5" 2>/dev/null)

            if [ -z "$latency" ]; then
                echo -n " N/A |" >> $OUTPUT_FILE
            else
                echo -n " ${latency} |" >> $OUTPUT_FILE
            fi
        fi
    done
    echo "" >> $OUTPUT_FILE
done

echo "" >> $OUTPUT_FILE
echo "## Analysis" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "- **Intra-cloud (AWS-AWS)**: Expected <5ms" >> $OUTPUT_FILE
echo "- **Intra-cloud (GCP-GCP)**: Expected <5ms" >> $OUTPUT_FILE
echo "- **Cross-cloud (AWS-GCP)**: Expected 50-100ms" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "## Performance Impact" >> $OUTPUT_FILE
echo "" >> $OUTPUT_FILE
echo "Based on the Raft consensus algorithm:" >> $OUTPUT_FILE
echo "- Leader election timeout: 3s (sufficient for cross-cloud)" >> $OUTPUT_FILE
echo "- Heartbeat interval: 1s (adequate given latencies)" >> $OUTPUT_FILE
echo "- Log replication: Latency directly affects write performance" >> $OUTPUT_FILE

echo "✅ Latency measurements saved to $OUTPUT_FILE"
cat $OUTPUT_FILE
