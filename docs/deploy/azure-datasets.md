# Deploy Azure — Datasets com cache persistente

Como rodar o mcp-brasil no Azure Container Apps com os datasets grandes
(SIAPA, TSE 2014-2024) pré-carregados e persistidos entre restarts/scale.

## Arquitetura

```
                ┌─────────────────────┐
                │ Azure Container App │
                │  mcp-brasil         │
                │  ENTRYPOINT:        │
                │  1. warmup datasets │
                │  2. start server    │
                └──────────┬──────────┘
                           │ SMB mount
                           │
               ┌───────────▼───────────┐
               │  Azure Files share    │
               │  mcp-brasil-cache     │
               │  (persistente 100GB)  │
               └───────────────────────┘
```

- **Entrypoint** (`docker-entrypoint.sh`): roda `scripts/warmup_datasets.py`
  antes de iniciar o servidor. Warmup baixa + ingere todos os datasets em
  `MCP_BRASIL_DATASETS`.
- **Azure Files**: SMB mount em `/cache/mcp-brasil` dentro do container.
  Preserva os arquivos `.duckdb` entre restarts de replicas e deploys.
- **Traffic weight**: Container Apps só muda traffic para a nova revisão
  quando ela está healthy (probe do ingress), ou seja, **só depois que o
  warmup completou e o server está aceitando requests**.

## Setup inicial (via Azure CLI)

### 1. Storage Account + File Share

```bash
RG=rg-mcp-brasil
SA_NAME=mcpbrasildata$(date +%s | tail -c 6)
SHARE=mcp-brasil-cache

# Provider (primeira vez)
az provider register -n Microsoft.Storage
until [ "$(az provider show -n Microsoft.Storage --query registrationState -o tsv)" = "Registered" ]; do sleep 10; done

# Storage
az storage account create \
  --name $SA_NAME \
  -g $RG \
  --location eastus2 \
  --sku Standard_LRS \
  --kind StorageV2

# File share (100GB mínimo Standard tier)
KEY=$(az storage account keys list -n $SA_NAME -g $RG --query "[0].value" -o tsv)
az storage share create --name $SHARE --account-name $SA_NAME --account-key "$KEY" --quota 100
```

### 2. Linkar ao Container App environment

```bash
ENV=mcp-brasil-env
STORAGE_NAME=mcp-cache    # identificador lógico dentro do env

az containerapp env storage set \
  -g $RG -n $ENV \
  --storage-name $STORAGE_NAME \
  --azure-file-account-name $SA_NAME \
  --azure-file-account-key "$KEY" \
  --azure-file-share-name $SHARE \
  --access-mode ReadWrite
```

### 3. Montar volume + configurar env + escalar recursos

Exportar o spec atual, patchar com Python, aplicar:

```bash
az containerapp show -n mcp-brasil -g $RG -o yaml > /tmp/app.yaml

python3 <<'PY'
import yaml, pathlib
doc = yaml.safe_load(pathlib.Path('/tmp/app.yaml').read_text())
c = doc['properties']['template']['containers'][0]

# env vars
envs = {e['name']: e for e in c['env']}
envs.update({k: {'name': k, 'value': v} for k, v in {
    'MCP_BRASIL_DATASETS': 'spu_siapa,tse_candidatos,tse_bens,tse_redes_sociais,tse_fefc,tse_votacao',
    'MCP_BRASIL_DATASET_CACHE_DIR': '/cache/mcp-brasil',
    'MCP_BRASIL_DATASET_TIMEOUT': '1200',
    'MCP_BRASIL_DATASET_REFRESH': 'auto',
}.items()})
c['env'] = list(envs.values())

# resources — tse_votacao alone needs ~3-4GB peak during ingestion
c['resources'] = {'cpu': 3.0, 'memory': '6Gi', 'ephemeralStorage': '8Gi'}

# volume mount
c['volumeMounts'] = [{'volumeName': 'dataset-cache', 'mountPath': '/cache/mcp-brasil'}]
doc['properties']['template']['volumes'] = [{
    'name': 'dataset-cache', 'storageType': 'AzureFile', 'storageName': 'mcp-cache',
}]

pathlib.Path('/tmp/app-patched.yaml').write_text(yaml.safe_dump(doc, sort_keys=False))
PY

az containerapp update -n mcp-brasil -g $RG --yaml /tmp/app-patched.yaml
```

### 4. Build imagem com warmup + deploy

```bash
# Build remoto via ACR Tasks (sem Docker local)
az acr build \
  --registry mcpbrasilacra9d3b3 \
  --image mcp-brasil:v0.14.0 \
  --image mcp-brasil:latest \
  --file Dockerfile .

# Apontar container app para a nova tag
az containerapp update \
  -n mcp-brasil -g $RG \
  --image mcpbrasilacra9d3b3.azurecr.io/mcp-brasil:v0.14.0
```

Ao subir uma nova revisão, o entrypoint dispara `warmup_datasets.py`:

```
[entrypoint] MCP_BRASIL_DATASETS=spu_siapa,tse_candidatos,...
[entrypoint] Cache dir: /cache/mcp-brasil
[entrypoint] Running warmup — this may take several minutes on first boot.
[warmup] Warmup plan: 6 datasets
[warmup] [spu_siapa] starting (approx 220 MB)...
[warmup] [spu_siapa] ok in 16.8s — 812,868 rows, 45.8 MB
[warmup] [tse_bens] starting (approx 205 MB)...
 ...
[warmup] Warmup complete: 6 ok, 0 failed, total XXX.Xs
[entrypoint] Starting MCP server on :8061
```

No **primeiro** boot: warmup baixa tudo (~15-30min total, `tse_votacao` é
1.6GB). Em deploys subsequentes: cache hit instantâneo no volume Azure Files.

## Monitorar

```bash
# Logs em tempo real
az containerapp logs show -n mcp-brasil -g rg-mcp-brasil --follow --tail 100

# Estado do volume
az storage file list --share-name mcp-brasil-cache \
  --account-name mcpbrasildata51273 --path datasets \
  --query "[].{name:name, size:properties.contentLength}" -o table

# Exec dentro do container
az containerapp exec -n mcp-brasil -g rg-mcp-brasil --command bash
```

## Troubleshooting

**Warmup OOM-killed durante `tse_votacao`** — dataset maior (1.6GB). Bump
memory para 6-8GB ou remova `tse_votacao` de `MCP_BRASIL_DATASETS` para
carga lazy.

**Warmup falha mas server sobe** — o entrypoint tem fallback: em caso de
erro, continua para `mcp.run(...)`. Datasets carregam sob demanda no
primeiro tool call (com timeout maior).

**Cache volume cheio** — `quota 100` é GB. Aumente via:
```bash
az storage share update --name mcp-brasil-cache \
  --account-name mcpbrasildata51273 --account-key "$KEY" --quota 500
```

**Reset total** — remover arquivos do share + redeploy:
```bash
az storage file delete-batch --source mcp-brasil-cache \
  --account-name mcpbrasildata51273 --account-key "$KEY"
az containerapp update -n mcp-brasil -g rg-mcp-brasil --revision-suffix refresh$(date +%s)
```

## Variáveis de ambiente relevantes

| Variável | Valor Azure | Descrição |
|---|---|---|
| `MCP_BRASIL_DATASETS` | `spu_siapa,tse_candidatos,tse_bens,tse_redes_sociais,tse_fefc,tse_votacao` | Datasets a warmup |
| `MCP_BRASIL_DATASET_CACHE_DIR` | `/cache/mcp-brasil` | Mount path do volume Azure Files |
| `MCP_BRASIL_DATASET_TIMEOUT` | `1200` | Timeout em segundos do download (20min) |
| `MCP_BRASIL_DATASET_REFRESH` | `auto` | Respeita TTL 30d por dataset |

## Custo aproximado

- Storage Account Standard_LRS: ~R$ 2-5/mês para 5GB de cache ativo
- File Share 100GB reserva: ~R$ 30/mês (parcial se usar pouco)
- Tráfego egress durante warmup: baixado do CDN TSE/SPU, sem custo extra
- Container App Consumption: paga CPU/mem só durante execução de requests
  + warmup. Warmup de 20min custa poucos centavos.

Total estimado para dev: **R$ 30-50/mês**.

## Limitações

- Azure Container Apps Consumption plan tem limites de CPU (4 vCPU) e
  memory (8GB) por container. Se precisar mais, migre para Dedicated
  (custos diferentes).
- Azure Files SMB tem latência maior que disco local — queries DuckDB em
  arquivos grandes têm overhead de ~50-200ms comparado a ephemeral. Aceito
  para persistência.
- Warmup sequencial (um dataset por vez). Para paralelizar, edite
  `scripts/warmup_datasets.py` usando `asyncio.gather` (cuidado com
  memória).

## Notas técnicas — DuckDB e SMB

DuckDB usa `mmap()` e `fcntl()` para crescer seu arquivo de banco. Essas
operações **não funcionam direto em SMB shares** (Azure Files), gerando
erros opacos como `IO Error: Cannot open file ... No such file or directory`
logo no `duckdb.connect(path, read_only=False)`.

A solução (implementada em `_shared/datasets/loader.py`): escrever o
`.duckdb` em `/tmp/` (tmpfs ephemeral) durante a ingestão, e só no final
usar `shutil.move()` para movê-lo ao cache Azure Files. Arquivos já
finalizados funcionam em read-only na SMB share normalmente — é só a
**criação/crescimento** que exige ephemeral.

Consequência prática: você precisa de **espaço ephemeral >= tamanho
máximo de um dataset em DuckDB**. Para `tse_votacao` (dataset maior),
reserve ~6-8GB de ephemeral storage no Container App.
