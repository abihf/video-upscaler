# video-upscaler

`video-upscaler` is a Go CLI and Temporal worker for queueing and processing anime/video upscaling jobs (1080p ➜ 2160p).

## Commands

### Queue a single file

```bash
video-upscaler add [-p priority] [-f] input-file.mkv [output-file.mkv]
```

- `-p, --priority`: `default`, `high`, or `low`
- `-f, --force`: terminate an existing workflow with the same output target before re-adding
- If `output-file.mkv` is omitted, the tool replaces the first `1080p` in the input filename with `2160p`

### Queue files recursively from a directory

```bash
video-upscaler scan /path/to/library
```

The scanner looks for `.mkv` files containing `SxxEyy` in the filename, then enqueues an upscale job when it finds a `1080p` episode without a matching `2160p` file.

### Run worker

```bash
video-upscaler worker
```

The worker consumes jobs from the Temporal task queue `upscaler`.

## Configuration

- `TEMPORAL_ADDRESS` (default: `localhost:7233`)
  - Temporal frontend endpoint used by both CLI commands and worker
- Temporal namespace is fixed to `upscaler`

## Build and test

From repository root:

```bash
go test ./...
go build ./...
```

## Container image

The repository Dockerfile supports:

- `cli` image target for command execution
- `worker` image target (default runtime command: `video-upscaler worker`)
