#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# Verify cross-cloud connectivity for Raft cluster

# Ensure jq is installed
if ! command -v jq &> /dev/null
then
    echo "ERROR: jq is not installed. Please install it (e.g., sudo apt-get install jq)."
    exit 1
fi

# Ensure terraform is installed and initialized
if ! command -v terraform &> /dev/null
then
    echo "ERROR: terraform is not installed. Please install it."
    exit 1
fi

# Check if terraform is initialized in the current directory
if [ ! -d ".terraform" ]; then
    echo "ERROR: Terraform is not initialized in the current directory. Please run 'terraform init'."
    exit 1
fi

SSH_KEY="~/.ssh/${KEY_NAME}.pem"

echo "=== Cross-Cloud Connectivity Verification ==="
echo ""

# Get Terraform outputs in JSON format
echo "Fetching IP addresses from Terraform outputs..."
TERRAFORM_OUTPUT=$(terraform output -json)

# Extract IPs using jq
# Use @json to ensure proper array parsing, then .[] to get individual elements
readarray -t AWS_IPS_ARRAY < <(echo "$TERRAFORM_OUTPUT" | jq -r '.aws_instance_public_ips.value | .[]')
readarray -t GCP_IPS_ARRAY < <(echo "$TERRAFORM_OUTPUT" | jq -r '.gcp_instance_public_ips.value | .[]')

# Combine all IPs into a single array for easier iteration
ALL_IPS_ARRAY=("${AWS_IPS_ARRAY[@]}" "${GCP_IPS_ARRAY[@]}")

echo "AWS IPs: ${AWS_IPS_ARRAY[*]}"
echo "GCP IPs: ${GCP_IPS_ARRAY[*]}"
echo "All IPs: ${ALL_IPS_ARRAY[*]}"
echo ""

echo "Testing connectivity from each node to all others..."
echo ""

for source_ip in "${ALL_IPS_ARRAY[@]}"; do
    # Determine SSH user based on IP
    IS_AWS=0
    for aws_ip in "${AWS_IPS_ARRAY[@]}"; do
        if [[ "$source_ip" == "$aws_ip" ]]; then
            IS_AWS=1
            break
        fi
    done

    if [[ "$IS_AWS" -eq 1 ]]; then
        SSH_USER="ec2-user"
    else
        SSH_USER="${GCP_USER_NAME}" # GCP_USER_NAME should be set by export
    fi

    echo "--- Testing from $source_ip ($SSH_USER) ---"

    for target_ip in "${ALL_IPS_ARRAY[@]}"; do
        if [ "$source_ip" = "$target_ip" ]; then
            continue
        fi

        # Test ping (ICMP)
        ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$SSH_USER@$source_ip" \
            "ping -c 3 -W 2 $target_ip > /dev/null 2>&1 && echo '✅ PING $target_ip' || echo '❌ PING $target_ip'"

        # Test TCP connection to app port (8080)
        ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$SSH_USER@$source_ip" \
            "timeout 5 bash -c 'cat < /dev/null > /dev/tcp/$target_ip/8080' 2>/dev/null && echo '✅ TCP $target_ip:8080' || echo '⚠️  TCP $target_ip:8080 (no listener yet)'"
    done
    echo ""
done

echo "=== Connectivity Test Complete ==="



# #!/bin/bash
# set -x
# # Verify cross-cloud connectivity for Raft cluster

# INVENTORY_FILE="inventory.ini"
# SSH_KEY="~/.ssh/${KEY_NAME}.pem"

# echo "=== Cross-Cloud Connectivity Verification ==="
# echo ""

# # Check if inventory file exists
# if [ ! -f "$INVENTORY_FILE" ]; then
#     echo "ERROR: inventory.ini not found. Please ensure Ansible inventory is generated."
#     echo "Run: terraform output to see instance IPs and create inventory.ini"
#     exit 1
# fi

# # Extract IPs from inventory
# AWS_IPS=$(awk '/^\[aws\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)
# echo "AWS_IPS: $AWS_IPS"
# GCP_IPS=$(awk '/^\[gcp\]/,/^\[/ {if ($0 !~ /^\[/ && $0 !~ /^$/) print $0}' $INVENTORY_FILE)
# echo "GCP_IPS: $GCP_IPS"
# ALL_IPS="$AWS_IPS $GCP_IPS"
# echo "ALL_IPS: $ALL_IPS"

# echo "Testing connectivity from each node to all others..."
# echo ""

# for source_ip in $ALL_IPS; do
#     # Determine SSH user based on IP
#     if echo "$AWS_IPS" | grep -q "$source_ip"; then
#         SSH_USER="ec2-user"
#     else
#         SSH_USER="${GCP_USER_NAME:-yourusername}"
#     fi

#     echo "--- Testing from $source_ip ($SSH_USER) ---"

#     for target_ip in $ALL_IPS; do
#         if [ "$source_ip" = "$target_ip" ]; then
#             continue
#         fi

#         # Test ping (ICMP)
#         ssh -i $SSH_KEY -o StrictHostKeyChecking=no $SSH_USER@$source_ip \
#             "ping -c 3 -W 2 $target_ip > /dev/null 2>&1 && echo '✅ PING $target_ip' || echo '❌ PING $target_ip'"

#         # Test TCP connection to app port (8080)
#         ssh -i $SSH_KEY -o StrictHostKeyChecking=no $SSH_USER@$source_ip \
#             "timeout 5 bash -c 'cat < /dev/null > /dev/tcp/$target_ip/8080' 2>/dev/null && echo '✅ TCP $target_ip:8080' || echo '⚠️  TCP $target_ip:8080 (no listener yet)'"
#     done
#     echo ""
# done

# echo "=== Connectivity Test Complete ==="
