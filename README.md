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

## Workflow

1. `video-upscaler add` (single file) or `video-upscaler scan` (library recursion) enqueues Temporal workflow `Upscale`.
2. The worker executes activities in order:
   - `Prepare`: create a per-workflow temp directory
   - `Info`: collect source media metadata
   - `Upscale`: run VapourSynth (`vspipe`) + encode with `ffmpeg`
   - `Merge`: mux upscaled video with original non-video streams via `mkvmerge`
   - `MoveFile`: move final MKV to target output path
   - `Delete`: cleanup temporary upscaled intermediate
3. The queue name is `upscaler` and namespace is `upscaler`.

## VapourSynth `script.py`

`script.py` is the frame-processing pipeline used by `vspipe` during the `Upscale` activity.

At a high level it:

1. Loads the input clip from `in` (with cache path `cache`).
2. Converts source to RGB half-float (`RGBH`) in Rec.709.
3. Runs TensorRT-backed upscaling via `vsmlrt`:
   - Default path: `RealESRGANv2` with model `animejanaiV3_HD_L2`
   - Optional custom model path inference
4. Optionally applies RIFE interpolation when enabled.
5. Converts processed frames to `YUV420P10` (Rec.709) and outputs to `vspipe`.

Environment variables used by `script.py`:

- `VSPIPE_NUM_STREAMS` (default `2`): TensorRT stream count for main upscaler backend
- `VISPIPE_MODEL_PATH`: optional explicit model path for `vsmlrt.inference` (this exact spelling is what `script.py` currently reads)
- `VSPIPE_MODEL_NAME` (default `animejanaiV3_HD_L2`): RealESRGANv2 model enum name
- `VSPIPE_RIFE` (default `0`): set to `1` to enable RIFE interpolation
- `VSPIPE_RIFE_MODEL` (default `v4_7`): RIFE model selection
- `VSPIPE_RIFE_NUM_STREAMS` (default `1`): stream count for RIFE backend

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

## Base image software (`base-image/Dockerfile`)

The `base-image` build assembles the runtime dependencies used by the `worker` image.

Main software components:

- **OS/base tooling**: Ubuntu 24.04, build-essential, clang/lld 22 toolchain
- **VapourSynth stack**:
  - VapourSynth `R73`
  - BestSource plugin `R16` (source filter)
  - (Akarin plugin build stage exists but is currently not copied into runtime)
- **ML inference stack**:
  - `vs-mlrt` (pinned git commit)
  - CUDA NVCC `13.1`
  - TensorRT RTX `1.3.0.35`
- **Video/mux tooling**:
  - FFmpeg `n8.0.1` built with NVENC/NVDEC and zimg/theora/opus/vpx support
  - `mkvtoolnix` for MKV muxing

Runtime shared libraries installed include Python 3.12 runtime and codec/misc libs used by FFmpeg/VapourSynth (`libtheora`, `libzimg`, `libopus`, `libvpx`, `libxxhash`).
