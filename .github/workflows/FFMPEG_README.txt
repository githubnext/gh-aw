# FFmpeg Agentic Workflows

This directory contains agentic workflows and shared configurations for using FFmpeg in GitHub Actions.

## Files Created

### 1. Shared Configuration
- **`.github/workflows/shared/ffmpeg-setup.md`** - Comprehensive guide for installing and using FFmpeg in workflows
  - Installation instructions using `FedericoCarboni/setup-ffmpeg` action or apt-get
  - Complete reference for common FFmpeg operations
  - Best practices extracted from GenAIScript implementation
  - Tips and tricks for video/audio processing

### 2. Example Workflows

#### Video Analyzer (`video-analyzer.md`)
A comprehensive video analysis workflow that can:
- Extract keyframes (I-frames) from videos
- Detect scene changes with configurable thresholds
- Extract audio in multiple formats
- Perform full video analysis with detailed reports
- Create GitHub issues with analysis results

**Usage:**
```bash
gh workflow run video-analyzer.md \
  --field video_url="https://example.com/video.mp4" \
  --field analysis_type="full_analysis"
```

**Analysis Types:**
- `keyframes` - Extract all keyframes
- `scenes` - Detect scene changes
- `audio_extract` - Extract audio tracks
- `full_analysis` - Perform all analyses

#### Audio Extractor (`audio-extractor.md`)
A simple audio extraction workflow that:
- Downloads videos from URLs
- Extracts audio in MP3, WAV, or AAC format
- Configurable quality settings
- Saves output as workflow artifacts

**Usage:**
```bash
gh workflow run audio-extractor.md \
  --field video_url="https://example.com/video.mp4" \
  --field audio_format="mp3" \
  --field quality="192k"
```

## Key Features Extracted from GenAIScript

Based on the [GenAIScript ffmpeg implementation](https://github.com/microsoft/genaiscript/blob/main/packages/core/src/ffmpeg.ts), these workflows incorporate the following best practices:

### 1. Keyframe Extraction
- Uses `select='eq(pict_type,I)'` to extract I-frames (keyframes)
- Variable frame rate mode (`-fps_mode vfr`) for proper keyframe extraction
- Frame presentation timestamps included (`-frame_pts 1`)

### 2. Scene Detection
- Uses `select='gt(scene,threshold)'` filter for detecting scene changes
- Threshold range: 0.0-1.0 (default: 0.4)
- Lower threshold = more sensitive (0.1-0.2 for minor changes)
- Higher threshold = less sensitive (0.5-0.6 for major changes only)
- Includes `showinfo` filter for timestamp logging

### 3. Audio Extraction
- Uses `-vn` flag to exclude video stream
- Supports multiple codecs: MP3 (libmp3lame), WAV (pcm_s16le), AAC
- Configurable bitrate with `-ab` flag
- Quality recommendations:
  - 128k for speech/podcasts
  - 192k for music (good balance)
  - 256k for high quality music
  - 320k for maximum quality MP3

### 4. Performance Best Practices
- **Caching**: Store processed results to avoid reprocessing
- **Concurrency limiting**: Process one video at a time (`pLimit(1)`)
- **Format selection**: JPG for frames (good quality/size balance)
- **Size control**: Use `.size()` or `-vf scale=` for output dimensions
- **Hash-based caching**: Use content hashing for cache keys

### 5. Quality Control
- CRF (Constant Rate Factor) for quality control (23 is good balance)
- Preset options: ultrafast → veryslow (speed vs quality tradeoff)
- Two-pass encoding for better quality at target bitrate
- Hardware acceleration when available (`-hwaccel auto`)

### 6. Error Handling
- Verify input files exist and are valid
- Check ffmpeg command success (exit code 0)
- Validate output files are not empty
- Proper cleanup of temporary files

## Usage in Your Workflows

To use the shared FFmpeg setup in your own workflows:

```yaml
---
on:
  workflow_dispatch:

permissions:
  contents: read

engine: copilot

imports:
  - shared/ffmpeg-setup.md

tools:
  bash:
---

# Your Workflow

You can now use ffmpeg commands as documented in the shared guide.

## Step 1: Install FFmpeg
Install ffmpeg first:
```bash
sudo apt-get update && sudo apt-get install -y ffmpeg
```

## Step 2: Use FFmpeg
Follow the examples in the imported guide for common operations.
```

## Installation Methods

### Method 1: Using apt-get (Recommended for Copilot/Claude/Codex engines)
```bash
sudo apt-get update && sudo apt-get install -y ffmpeg
ffmpeg -version
```

### Method 2: Using FedericoCarboni/setup-ffmpeg (For custom engines)
```yaml
steps:
  - name: Setup FFmpeg
    uses: FedericoCarboni/setup-ffmpeg@v3
    with:
      ffmpeg-version: release
```

## Common FFmpeg Operations

The shared guide includes detailed examples for:
- Audio extraction (MP3, WAV, AAC)
- Keyframe extraction
- Scene detection
- Video resizing and conversion
- Timestamp-based frame extraction
- Quality optimization
- Error handling
- Performance tuning

## References

- [FFmpeg Official Documentation](https://ffmpeg.org/documentation.html)
- [FFmpeg Wiki](https://trac.ffmpeg.org/wiki)
- [GenAIScript FFmpeg Implementation](https://github.com/microsoft/genaiscript/blob/main/packages/core/src/ffmpeg.ts)
- [FedericoCarboni/setup-ffmpeg Action](https://github.com/FedericoCarboni/setup-ffmpeg)

## License

These workflows are part of the GitHub Agentic Workflows project.
