#!/bin/bash

# install project dependencies
make deps

# configure the Vertex API access
# take vertex_api_json from environment variable and write it to `.credentials/vertex_api.json`
if [ -z "$VERTEX_API_JSON" ]; then
  echo "VERTEX_API_JSON environment variable is not set. Please set it to your Vertex API JSON credentials."
  exit 0
fi  
mkdir -p .credentials
echo "$VERTEX_API_JSON" > .credentials/vertex_api.json


#install gcloud CLI
sudo apt-get install apt-transport-https ca-certificates gnupg curl -y
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
sudo apt-get update && sudo apt-get install google-cloud-cli -y

export GOOGLE_APPLICATION_CREDENTIALS=.credentials/vertex_api.json
export CLAUDE_CODE_USE_VERTEX=1
export CLOUD_ML_REGION=us-east5
export ANTHROPIC_VERTEX_PROJECT_ID=github-next

# install uvx
curl -LsSf https://astral.sh/uv/install.sh | sh