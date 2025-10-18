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

steps:
  - name: Setup FFmpeg
    run: |
      sudo apt-get update && sudo apt-get install -y ffmpeg
      ffmpeg -version
---

# FFmpeg Usage Guide

FFmpeg has been installed and is available in your PATH.

## Finding FFmpeg

You can verify ffmpeg installation and get information:

```bash
# Check ffmpeg version and installation
ffmpeg -version

# Get ffmpeg location
which ffmpeg

# Get ffprobe (companion tool) location
which ffprobe
```

## Common FFmpeg Operations

### Extract Audio from Video

```bash
# Extract audio as MP3 with high quality
ffmpeg -i input.mp4 -vn -acodec libmp3lame -ab 192k output.mp3

# Extract audio as WAV (uncompressed)
ffmpeg -i input.mp4 -vn -acodec pcm_s16le output.wav

# Extract audio as AAC
ffmpeg -i input.mp4 -vn -acodec aac -ab 128k output.aac

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

## References

- FFmpeg Official Documentation: https://ffmpeg.org/documentation.html
- FFmpeg Wiki: https://trac.ffmpeg.org/wiki
- Opus Codec Documentation: https://opus-codec.org/

