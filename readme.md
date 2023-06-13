# Very low latency basic audio PCM S16LE UDP streamer

UDP should work fine through a single switch where there is virtually no possibility of the packets arriving out of order and the PCM format is known

To play the stream sent by audio_streamer_send, either use audio_streamer_receive or:
```ffplay -f s16le -ac 2 -ar 48000 "udp://224.0.0.3:1234"```
