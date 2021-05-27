# HighLite

Share Shadowplay game clips over a Discord Webhook. Free-tier users can share their game clips, as ffmpeg compresses the clips to meet 8MB file size limit requirement. This program watches for changes in the Shadowplay clip folder. When a clip is added, HighLite compresses it, and POSTs to the Discord Webhook specified in webhook.txt. 

# Usage

- Rename webhook-example.txt to webhook.txt with your discord webhook
- Run HighLite.exe

# Compatability 
HighLite was developed for the intention of Windows. NVIDIA GPU's are much more common on Windows machines and have better compatability than Linux, so Windows Registry is leveraged. 
