#!/bin/bash

# Script to delete all Go files in internal/db and internal/repository directories

echo "Deleting Go files in internal/db..."
find internal/db -name "*.go" -type f -delete

echo "Deleting Go files in internal/repository..."
find internal/repository -name "*.go" -type f -delete

echo "Cleanup complete!"