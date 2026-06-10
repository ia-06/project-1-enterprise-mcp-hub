#!/bin/bash
echo "========================================================"
echo "Enterprise MCP Hub: Compiling Go Backend for Native MCP"
echo "========================================================"
echo ""
echo "Building the Go binary..."

cd mcp-server
go build -o mcp-hub ./cmd/server

if [ $? -ne 0 ]; then
    echo "[ERROR] Build failed! Please check the Go compiler output."
    exit 1
fi

echo "[SUCCESS] Binary compiled successfully: mcp-server/mcp-hub"
echo ""
echo "IMPORTANT: Point your IDE's MCP Configuration to this executable."
echo "Do NOT use 'go run' in your IDE configuration as stdout compiler"
echo "messages will fatally corrupt the JSON-RPC protocol."
echo ""
cd ..
