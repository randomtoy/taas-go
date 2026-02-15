# tarot-as-a-service

HTTP service that generates tarot spreads with neutral LLM interpretations via OpenRouter.

## Quick start

```bash
# Set required env var
export OPENROUTER_API_KEY=your-key-here

# Run locally
make run

# Run tests
make test

# Build Docker image
make docker
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Server listen address |
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `LLM_PROVIDER` | `openrouter` | LLM provider |
| `LLM_MODEL` | `qwen/qwen3-4b:free` | Model identifier |
| `LLM_FALLBACK_MODELS` | *(empty)* | Comma-separated fallback model IDs (tried in order if primary fails) |
| `OPENROUTER_API_KEY` | *(required)* | OpenRouter API key |
| `OPENROUTER_BASE_URL` | `https://openrouter.ai/api/v1` | OpenRouter base URL |
| `LLM_TIMEOUT` | `10s` | Timeout for LLM requests |

## API

### GET /healthz

```bash
curl http://localhost:8080/healthz
# OK
```

### GET /v1/tarot

Generate a tarot spread with LLM interpretation.

**Parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | *(empty)* | Optional question (max 500 chars) |
| `n` | int | `3` | Number of cards (1-10) |
| `deck` | string | `major_arcana` | Deck ID |
| `spread` | string | `generic` | Spread type |
| `lang` | string | `en` | Interpretation language (BCP 47 code, e.g. `ru`, `es`, `fr`) |

**Examples:**

```bash
# Default 3-card spread
curl "http://localhost:8080/v1/tarot"

# With a question
curl "http://localhost:8080/v1/tarot?q=What+should+I+focus+on+today%3F"

# 5-card spread
curl "http://localhost:8080/v1/tarot?n=5&q=Career+outlook"
```

**Response (200):**

```json
{
  "spread": "three_card",
  "deck": "major_arcana",
  "cards": [
    {
      "id": "the_fool",
      "name": "The Fool",
      "position": 1,
      "orientation": "upright",
      "keywords": ["beginnings", "spontaneity", "trust", "innocence"],
      "short": "A fresh start and openness to experience."
    }
  ],
  "interpretation": {
    "style": "neutral",
    "text": "...",
    "disclaimer": "For reflection/entertainment; not medical/legal/financial advice."
  },
  "meta": {
    "model": "qwen/qwen3-4b:free",
    "request_id": "abc123",
    "latency_ms": 1234
  }
}
```

## Project structure

```
cmd/tarotd/              Main entrypoint
internal/
  domain/                Domain models and pure logic
  ports/                 Interfaces (RNG, DeckStore, Interpreter)
  app/                   Application use-cases
  adapters/
    http/                Echo handlers, middleware, DTOs
    llm/openrouter/      OpenRouter LLM adapter
    decks/               Embedded deck data store
  config/                Configuration
api/                     OpenAPI spec
deploy/helm/             Helm chart for k3s
.github/workflows/       CI, Release, Deploy pipelines
```

## Available decks

- `major_arcana` — 22 Major Arcana cards

Architecture supports adding more decks (e.g., `rws_78`, `thoth_78`) as embedded JSON files in `internal/adapters/decks/data/`.

## CI/CD

Three GitHub Actions workflows (matching radiomap-backend style):

### CI (`ci.yml`)
Runs on push/PR to `main`:
- `go test -v -race` with coverage
- `golangci-lint`
- Docker build (no push)
- Helm lint + template

### Release (`release.yml`)
Triggered by version tags (`v*.*.*`):
- Builds multi-arch Docker image (amd64 + arm64)
- Pushes to `ghcr.io/randomtoy/taas-go:<version>` and `:latest`
- Packages and pushes Helm chart to `oci://ghcr.io/randomtoy/charts`

### Deploy (`deploy.yml`)
Triggered automatically after Release or manually via `workflow_dispatch`:
- Establishes SSH tunnel to private k3s cluster
- Creates namespace and app secrets
- Runs `helm upgrade --install --atomic` with `--wait`
- Verifies rollout

**Release flow:**
```bash
git tag v1.0.0
git push origin v1.0.0
# Release workflow -> Docker + Helm to GHCR
# Deploy workflow -> SSH tunnel -> helm upgrade on k3s
```

## Kubernetes deployment

### Helm chart

```bash
# Local template check
helm template test deploy/helm/tarot-as-a-service --debug

# Manual install (if tunnel is already open)
helm upgrade --install tarot-as-a-service deploy/helm/tarot-as-a-service \
  --namespace tarot --create-namespace \
  --set image.tag=1.0.0
```

### Create app secret manually (if not using CI)

```bash
kubectl create secret generic tarot-app-secrets \
  --namespace tarot \
  --from-literal=OPENROUTER_API_KEY=sk-or-your-key
```

### Create GHCR image pull secret (for k3s)

```bash
kubectl create secret docker-registry ghcr-pull-secret \
  --namespace tarot \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USER \
  --docker-password=YOUR_GITHUB_PAT
```

Then add to values:
```yaml
imagePullSecrets:
  - name: ghcr-pull-secret
```

## GitHub configuration

### Secrets (required)

Sensitive values — must be stored in **Settings > Secrets and variables > Actions > Secrets**.

| Secret | Description |
|---|---|
| `SSH_PRIVATE_KEY` | SSH private key for k3s deployment host |
| `KUBECONFIG_B64` | Base64-encoded kubeconfig (`cat kubeconfig \| base64 -w0`). Server URL should point to `127.0.0.1:<port>` — it will be patched to use the SSH tunnel |
| `OPENROUTER_API_KEY` | OpenRouter API key, injected as k8s Secret `tarot-app-secrets` |

### Secrets (optional)

| Secret | Default | Description |
|---|---|---|
| `HELM_VALUES_B64` | *(none)* | Base64-encoded custom `values.yaml` override for Helm. Allows overriding any chart value without committing to repo |

### Variables (required)

Non-sensitive config — store in **Settings > Secrets and variables > Actions > Variables**.

| Variable | Description |
|---|---|
| `SSH_HOST` | SSH hostname or IP for k3s access |
| `SSH_USER` | SSH username |

### Variables (optional)

| Variable | Default | Description |
|---|---|---|
| `SSH_PORT` | `22` | SSH port |
| `K8S_API_HOST` | `127.0.0.1` | k8s API server address as seen from SSH host |
| `K8S_API_PORT` | `6443` | k8s API server port |
| `HELM_RELEASE_NAME` | `tarot-as-a-service` | Helm release name |
| `HELM_NAMESPACE` | `tarot` | Kubernetes namespace |

### HELM_VALUES_B64 reference

`HELM_VALUES_B64` is a base64-encoded YAML file that overrides default chart values.
Only include the keys you want to change — everything else uses defaults from
[`values.yaml`](deploy/helm/tarot-as-a-service/values.yaml).

Example file with all available keys:
[`custom-values.example.yaml`](deploy/helm/tarot-as-a-service/custom-values.example.yaml)

```bash
# Copy, edit, encode, paste into GitHub Secret
cp deploy/helm/tarot-as-a-service/custom-values.example.yaml custom-values.yaml
# edit custom-values.yaml
cat custom-values.yaml | base64 -w0
# paste the output into GitHub Secret HELM_VALUES_B64
```
