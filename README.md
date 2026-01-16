# helm-set-status

A Helm plugin that sets the status of a release to any valid Helm status value.

## Installation

```bash
helm plugin install https://github.com/josegonzalez/helm-set-status
```

## Usage

```bash
helm set-status RELEASE STATUS [flags]
```

### Arguments

- `RELEASE`: Name of the release to modify
- `STATUS`: Target status (one of the valid values below)

### Flags

| Flag | Description |
|------|-------------|
| `--revision` | Update a specific revision instead of the latest (default: 0 = latest) |
| `--from` | Only change status if current status is one of these values (can specify multiple) |
| `--no-fail` | Exit 0 instead of 1 when `--from` precondition is not met (prints skip message) |

### Valid Status Values

| Status | Description |
|--------|-------------|
| `unknown` | Unknown status |
| `deployed` | Release is deployed |
| `superseded` | Release has been superseded by a newer version |
| `failed` | Release failed to deploy |
| `uninstalling` | Release is being uninstalled |
| `pending-install` | Release is pending installation |
| `pending-upgrade` | Release is pending upgrade |
| `pending-rollback` | Release is pending rollback |

### Environment Variables

The plugin respects standard Helm environment variables:

- `HELM_NAMESPACE`: Target namespace (default: "default")
- `HELM_KUBECONTEXT`: Kubernetes context to use
- `HELM_DRIVER`: Storage driver (default: secrets)
- `KUBECONFIG`: Kubernetes config file path

## Examples

```bash
# Set a release to failed status
helm set-status my-release failed

# Set a release back to deployed
helm set-status my-release deployed

# Set status in a specific namespace
HELM_NAMESPACE=production helm set-status my-release deployed

# Set status of a specific revision
helm set-status my-release failed --revision 3

# Fix a stuck old revision while keeping current revision intact
helm set-status my-release superseded --revision 1

# Only change to deployed if currently pending-upgrade or pending-rollback
helm set-status my-release deployed --from pending-upgrade --from pending-rollback

# Only change to failed if currently pending-install
helm set-status my-release failed --from pending-install

# Conditionally change status without failing if precondition doesn't match
helm set-status my-release deployed --from pending-upgrade --no-fail
# If current status is "failed": prints "Skipped: ...", exits 0
# If current status is "pending-upgrade": changes to deployed, exits 0
```

## Use Cases

- **Recovery from stuck releases**: Fix releases stuck in `pending-install`, `pending-upgrade`, or `pending-rollback` states
- **Testing**: Simulate different release states for testing purposes
- **Manual intervention**: Override release status when automatic processes fail
- **Safe automation**: Use `--from` to ensure status changes only happen when the release is in an expected state
- **Soft precondition checks**: Use `--no-fail` with `--from` to conditionally change status without treating mismatches as errors

## Building from Source

```bash
# Clone the repository
git clone https://github.com/josegonzalez/helm-set-status.git
cd helm-set-status

# Build
make build

# Run tests
make test

# Run tests with coverage
make cover

# Install locally
helm plugin install .
```

## License

MIT License - see [LICENSE](LICENSE) for details.
