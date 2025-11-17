#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

URL=$1

# Use curl to check the URL, suppressing output and following redirects.
# -s: Silent mode
# -o /dev/null: Discard output
# -w "%{http_code}": Output only the HTTP status code
# -L: Follow redirects
HTTP_CODE=$(curl -o /dev/null -s --head -w "%{http_code}" -L "$URL")

# Check if the HTTP code indicates success (e.g., 200 OK)
if [[ "$HTTP_CODE" -ge 200 && "$HTTP_CODE" -lt 300 ]]; then
  echo "{\"url_exists\": \"true\", \"http_code\": \"$HTTP_CODE\"}"
else
  echo "{\"url_exists\": \"false\", \"http_code\": \"$HTTP_CODE\"}"
fi