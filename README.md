# 🐳 Docker Parallel Pull

Pull multiple Docker images in parallel with retry logic and security features.

## 🚀 Usage

```bash
# Basic usage (uses config.yaml)
go run main.go

# Use custom config file
go run main.go custom-config.yaml
```

## ⚙️ Configuration

Configuration is managed through YAML files only. The application looks for `config.yaml` by default, or you can specify a custom config file as the first argument.

### 📁 Files

**containers.yaml**:
```yaml
images:
  - alpine:latest
  - nginx:stable
  - redis:7-alpine
```

**config.yaml**:
```yaml
container_file: "containers.yaml"
max_concurrency: 5
timeout: "5m"
max_retries: 3
retry_delay: "2s"
cleanup_after_test: true
show_pull_detail: false
show_progress: true
output_format: "text"
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `container_file` | `containers.yaml` | 📄 Container images file |
| `max_concurrency` | `5` | 🔄 Max concurrent pulls |
| `max_retries` | `3` | 🔁 Max retry attempts |
| `timeout` | `5m` | ⏱️ Timeout per pull |
| `retry_delay` | `2s` | ⏳ Base delay between retries |
| `output_format` | `text` | 📊 Output format (text/json) |
| `show_pull_detail` | `false` | 🔍 Show detailed output |
| `cleanup_after_test` | `true` | 🗑️ Remove images after pull |
| `show_progress` | `true` | 📈 Show progress bar |

## ✨ Features

- 🔄 Parallel image pulling with concurrency control
- 🔁 Exponential backoff retry logic
- 📈 Real-time progress tracking
- 🔒 Security validation (path traversal, input validation)
- 🛡️ Resource limits (file size, image count, timeouts)
- 📊 JSON and text output formats

## 📋 Requirements

- Go 1.25+
- Docker daemon running

## 📝 License

MIT
