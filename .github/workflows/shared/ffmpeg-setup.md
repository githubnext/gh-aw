---
# FFmpeg Setup Instructions
# Shared instructions for installing and using ffmpeg in workflows
#
# Usage:
#   imports:
#     - shared/ffmpeg-setup.md
#
# This import provides:
# - Instructions on how to install ffmpeg using FedericoCarboni/setup-ffmpeg action
# - Instructions on how to find and use ffmpeg
# - Best practices for video/audio processing
---

# FFmpeg Usage Guide

## Installing FFmpeg

FFmpeg is **not** pre-installed on GitHub Actions runners. You need to install it first using the `FedericoCarboni/setup-ffmpeg` action.

**Add this step to your workflow BEFORE using ffmpeg:**

If you're using a custom engine with explicit steps:
```yaml
steps:
  - name: Setup FFmpeg
    uses: FedericoCarboni/setup-ffmpeg@v3
    id: setup-ffmpeg
    with:
      ffmpeg-version: release
```

If you're using Copilot/Claude/Codex engines, you can install ffmpeg using bash:
```bash
# Install ffmpeg on Ubuntu runners
sudo apt-get update && sudo apt-get install -y ffmpeg

# Verify installation
ffmpeg -version
```

FFmpeg will be available in your PATH after installation.

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
```

**Key flags:**
- `-vn`: No video output
- `-acodec`: Audio codec (libmp3lame, pcm_s16le, aac)
- `-ab`: Audio bitrate (128k, 192k, 256k, 320k)

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

## Best Practices

### Performance Optimization

1. **Use hardware acceleration when available:**
   ```bash
   ffmpeg -hwaccel auto -i input.mp4 output.mp4
   ```

2. **Limit concurrent operations:**
   - Process videos one at a time for large files
   - Use batch processing for small files
   - Monitor memory usage on runners

3. **Cache processed results:**
   - Store processed outputs in workflow artifacts
   - Use GitHub Actions cache for intermediate results
   - Hash input files to detect changes

### Quality vs Speed Tradeoffs

1. **Fast encoding (lower quality):**
   ```bash
   ffmpeg -i input.mp4 -preset ultrafast -crf 28 output.mp4
   ```

2. **Slow encoding (higher quality):**
   ```bash
   ffmpeg -i input.mp4 -preset slow -crf 18 output.mp4
   ```

3. **Presets:** ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow

### Error Handling

```bash
# Check if ffmpeg command succeeded
if ffmpeg -i input.mp4 output.mp4; then
  echo "Success"
else
  echo "Failed with exit code $?"
  exit 1
fi

# Verify output file exists and is not empty
if [ ! -s output.mp4 ]; then
  echo "Output file is empty or missing"
  exit 1
fi
```

## Common Issues and Solutions

### Issue: "No such file or directory"
**Solution:** Use absolute paths or verify working directory

### Issue: "Unknown decoder" or "Codec not found"
**Solution:** Check available codecs with `ffmpeg -codecs` and use supported alternatives

### Issue: "Output file is empty"
**Solution:** Check input file format compatibility and verify ffmpeg command syntax

### Issue: "Conversion failed" with large files
**Solution:** 
- Increase memory limits for GitHub Actions runner
- Process video in smaller chunks
- Use lower quality settings
- Consider using two-pass encoding for better efficiency

## Advanced Techniques

### Two-Pass Encoding for Better Quality

```bash
# First pass
ffmpeg -i input.mp4 -c:v libx264 -b:v 2M -pass 1 -f null /dev/null

# Second pass
ffmpeg -i input.mp4 -c:v libx264 -b:v 2M -pass 2 output.mp4
```

### Extract Audio with Noise Reduction

```bash
# Apply high-pass filter to remove low-frequency noise
ffmpeg -i input.mp4 -vn -af "highpass=f=200" output.mp3

# Apply noise reduction with custom profile
ffmpeg -i input.mp4 -vn -af "anlmdn=s=10:p=0.002:r=0.002:m=15" output.mp3
```

### Timestamp-Based Frame Extraction

```bash
# Extract frames at specific timestamps (in seconds)
timestamps=(5.5 10.2 15.8 20.3)
for ts in "${timestamps[@]}"; do
  ffmpeg -ss "$ts" -i input.mp4 -frames:v 1 "frame_${ts}.jpg"
done
```

### Batch Processing with Progress

```bash
# Process multiple videos with progress tracking
total_files=$(ls *.mp4 | wc -l)
current=0

for video in *.mp4; do
  current=$((current + 1))
  echo "Processing $current/$total_files: $video"
  ffmpeg -i "$video" -c:v libx264 -crf 23 "processed_${video}"
done
```

## References

- FFmpeg Official Documentation: https://ffmpeg.org/documentation.html
- FFmpeg Wiki: https://trac.ffmpeg.org/wiki
- setup-ffmpeg Action: https://github.com/FedericoCarboni/setup-ffmpeg
