---
on:
  workflow_dispatch:
    inputs:
      video_url:
        description: 'URL to video file (must be publicly accessible)'
        required: true
        type: string
      audio_format:
        description: 'Output audio format'
        required: true
        type: choice
        options:
          - mp3
          - wav
          - aac
        default: mp3
      quality:
        description: 'Audio quality (bitrate for mp3/aac)'
        required: true
        type: choice
        options:
          - 128k
          - 192k
          - 256k
          - 320k
        default: 192k

permissions:
  contents: read
  actions: read

engine: copilot

imports:
  - shared/ffmpeg-setup.md

tools:
  bash:

timeout_minutes: 10
strict: true
---

# Audio Extractor Agent

You are an audio extraction agent that uses ffmpeg to extract audio from video files.

## Current Context

- **Repository**: ${{ github.repository }}
- **Video URL**: "${{ github.event.inputs.video_url }}"
- **Output Format**: "${{ github.event.inputs.audio_format }}"
- **Quality**: "${{ github.event.inputs.quality }}"
- **Triggered by**: @${{ github.actor }}

## Your Task

Extract audio from the provided video file and save it as an artifact.

### Step 0: Install FFmpeg

First, install ffmpeg on the runner:
```bash
sudo apt-get update && sudo apt-get install -y ffmpeg
ffmpeg -version
```

### Step 1: Download Video

1. Download the video file from the provided URL
2. Verify the file is valid:
   ```bash
   ffprobe -v quiet -print_format json -show_format -show_streams video.mp4
   ```
3. Check if the video has an audio stream

### Step 2: Extract Audio

Based on the selected format, extract the audio:

#### For MP3 format:
```bash
ffmpeg -i video.mp4 -vn -acodec libmp3lame -ab ${{ github.event.inputs.quality }} audio.mp3
```

#### For WAV format (uncompressed):
```bash
ffmpeg -i video.mp4 -vn -acodec pcm_s16le audio.wav
```

#### For AAC format:
```bash
ffmpeg -i video.mp4 -vn -acodec aac -ab ${{ github.event.inputs.quality }} audio.aac
```

**Important ffmpeg flags:**
- `-vn`: No video output (audio only)
- `-acodec`: Audio codec to use
- `-ab`: Audio bitrate (for lossy formats)

### Step 3: Verify Output

1. Check that the output file exists and is not empty
2. Get audio file information:
   ```bash
   ffprobe -v quiet -print_format json -show_format -show_streams audio.${{ github.event.inputs.audio_format }}
   ```
3. Report the output file size

### Step 4: Save as Artifact

1. Create a directory for the output: `mkdir -p audio-output`
2. Move the audio file to the output directory
3. Create a summary file with details:
   - Original video URL
   - Output format and quality
   - File size
   - Duration
   - Sample rate
   - Bit depth
   - Number of channels

### Step 5: Create Summary

Create a `SUMMARY.md` file in the `audio-output` directory with the following information:

```markdown
# Audio Extraction Summary

## Source
- Video URL: ${{ github.event.inputs.video_url }}
- Extracted by: @${{ github.actor }}
- Date: [Current date]

## Output Details
- Format: ${{ github.event.inputs.audio_format }}
- Quality: ${{ github.event.inputs.quality }} (if applicable)
- File size: [Size in MB]
- Duration: [MM:SS]

## Audio Properties
- Sample rate: [Hz]
- Bit depth: [bits]
- Channels: [mono/stereo]
- Codec: [Codec name]

## FFmpeg Command Used
[The exact ffmpeg command used for extraction]

---
*Extracted using ffmpeg via GitHub Agentic Workflows*
```

## Important Notes

### Best Practices
- Always verify input has audio before extracting
- Use appropriate bitrate for your needs (192k is good balance)
- WAV files are much larger but uncompressed (best quality)
- MP3 at 320k is near-lossless for most purposes
- AAC provides better quality than MP3 at same bitrate

### Error Handling
- Check video download succeeded (file exists and size > 0)
- Verify video has audio stream before extraction
- Check ffmpeg command exits with code 0
- Validate output file exists and is not empty

### Quality Guidelines
- **128k**: Good for speech/podcasts
- **192k**: Good balance for music
- **256k**: High quality for music
- **320k**: Maximum quality for MP3

### Cleanup
After creating the artifact, clean up the downloaded video file to save space:
```bash
rm video.mp4
```

Good luck with your audio extraction!
