---
# FFmpeg Setup
# Shared configuration for installing and using ffmpeg in workflows
#
# Usage:
#   imports:
#     - shared/ffmpeg.md
#
# This import provides:
# - Automatic ffmpeg installation via setup steps
# - Instructions on how to use ffmpeg
# - Best practices for video/audio processing
#
# Note: FFmpeg operations can be time-intensive. Ensure your workflow has
# adequate timeout-minutes (recommended: 15+ minutes for video processing).
# Individual bash commands using ffmpeg have a 5-minute timeout by default.

tools:
  bash:
    - "ffmpeg *"
    - "ffprobe *"

steps:
  - name: Setup FFmpeg
    id: setup-ffmpeg
    run: |
      sudo apt-get update && sudo apt-get install -y ffmpeg
      version=$(ffmpeg -version | head -n1)
      echo "version=$version" >> "$GITHUB_OUTPUT"
      mkdir -p /tmp/gh-aw/ffmpeg
---

# FFmpeg Usage Guide

FFmpeg and ffprobe have been installed and are available in your PATH. A temporary folder `/tmp/gh-aw/ffmpeg` is available for caching intermediate results.

**Note**: FFmpeg operations can take several minutes for large video files. Bash commands have a 5-minute timeout. For longer operations, break them into smaller steps or increase workflow timeout-minutes.

## Common FFmpeg Operations

### Extract Audio from Video

```bash
# Extract audio as MP3 with high quality
ffmpeg -i input.mp4 -vn -acodec libmp3lame -ab 192k output.mp3

# Extract audio for transcription (optimized for speech-to-text)
# Uses Opus codec with mono channel and low bitrate for optimal transcription
ffmpeg -i input.mp4 -vn -acodec libopus -ac 1 -ab 12k -application voip -map_metadata -1 -f ogg output.ogg
```

**Key flags:**
- `-vn`: No video output
- `-acodec`: Audio codec (libmp3lame, pcm_s16le, aac, libopus)
- `-ab`: Audio bitrate (128k, 192k, 256k, 320k, or 12k for transcription)
- `-ac`: Audio channels (1 for mono, 2 for stereo)
- `-application voip`: Optimize Opus for voice (for transcription)
- `-map_metadata -1`: Remove metadata

**For transcription:**
- Use `libopus` codec with OGG format
- Mono channel (`-ac 1`) is sufficient for speech
- Low bitrate (12k) keeps file size small
- `-application voip` optimizes for voice

### Extract Video Frames

```bash
# Extract all keyframes (I-frames)
ffmpeg -i input.mp4 -vf "select='eq(pict_type,I)'" -fps_mode vfr -frame_pts 1 keyframe_%06d.jpg

# Extract frames at specific interval (e.g., 1 frame per second)
ffmpeg -i input.mp4 -vf "fps=1" frame_%06d.jpg

# Extract single frame at specific timestamp
ffmpeg -i input.mp4 -ss 00:00:05 -frames:v 1 frame.jpg
```

**Key flags:**
- `-vf`: Video filter
- `-fps_mode vfr`: Variable frame rate (for keyframes)
- `-frame_pts 1`: Include frame presentation timestamp
- `-ss`: Seek to timestamp (HH:MM:SS or seconds)
- `-frames:v`: Number of video frames to extract

### Scene Detection

```bash
# Detect scene changes with threshold (0.0-1.0, default 0.4)
# Lower threshold = more sensitive to changes
ffmpeg -i input.mp4 -vf "select='gt(scene,0.3)',showinfo" -fps_mode passthrough -frame_pts 1 scene_%06d.jpg

# Common threshold values:
# 0.1-0.2: Very sensitive (minor changes trigger detection)
# 0.3-0.4: Moderate sensitivity (good for most videos)
# 0.5-0.6: Less sensitive (only major scene changes)
```

**Scene detection tips:**
- Start with threshold 0.4 and adjust based on results
- Use `showinfo` filter to see timestamps in logs
- Lower threshold detects more scenes but may include false positives
- Higher threshold misses gradual transitions

### Resize and Convert

```bash
# Resize video to specific dimensions (maintains aspect ratio)
ffmpeg -i input.mp4 -vf "scale=1280:720" output.mp4

# Resize with padding to maintain aspect ratio
ffmpeg -i input.mp4 -vf "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2" output.mp4

# Convert to different format with quality control
ffmpeg -i input.mp4 -c:v libx264 -crf 23 -c:a aac -b:a 128k output.mp4
```

**Quality flags:**
- `-crf`: Constant Rate Factor (0-51, lower=better quality, 23 is default)
- `18`: Visually lossless
- `23`: High quality (default)
- `28`: Medium quality

### Get Video Information

```bash
# Get detailed video information
ffprobe -v quiet -print_format json -show_format -show_streams input.mp4

# Get video duration
ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 input.mp4

# Get video dimensions
ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=s=x:p=0 input.mp4
```

### Compute Stable Hash for Video Encoding Task

Compute a SHA-256 hash that uniquely identifies an ffmpeg command and all input files it references. This is useful for caching and detecting when re-processing is needed.

**Steps:**
1. Capture the full ffmpeg command line (exact text with all arguments)
2. Concatenate the command string with the binary contents of each input file in the same order
3. Pipe the combined data into `sha256sum` (or `shasum -a 256` on macOS)

**Example Bash:**

```bash
cmd='ffmpeg -i input1.mp4 -i input2.wav -filter_complex "..." -c:v libx264 output.mp4'
(
  echo "$cmd"
  cat input1.mp4 input2.wav
) | sha256sum | awk '{print $1}'
```

This hash changes only when:
- The ffmpeg command arguments change
- Any input file content changes

Use this hash as a cache key in `/tmp/gh-aw/ffmpeg/` to avoid reprocessing identical operations.

