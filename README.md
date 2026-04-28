# Vendor
This repository vendors its Go dependencies under `vendor/` for reproducible/offline builds.

When using YCSB or SMALLBANK to send transactions, you must add the `-mod=vendor` flag to your Go run commands to use the vendored dependencies.

# Send YCSB PAYLOAD
Read the ycsb/README.md to send ycsb transactions.

# Send SMALLBANK PAYLOAD
Read the smallbank/README.md to send smallbank transactions.
