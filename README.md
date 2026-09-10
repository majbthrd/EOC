EOC (Ephemeral Ollama Client)
=============================

[Ollama](https://ollama.com) is a painless way to get a local LLM up and running.  It is multi-platform, it just works, and it is [open-source](https://github.com/ollama/ollama).

For Windows and macOS, Ollama even provides a multi-turn LLM chat GUI with a rich feature set (particularly in rendering).  An unofficial Linux adaptation is enabled by [this excellent effort](https://github.com/johnohhh1/ollama-webchat-ubuntu).

The great thing about open-source is that the software can be customized to suit the user's needs, and those changes can be shared back to the community.  The changes I made here were: a) disable Ollama cloud telemetry and downloads, and b) disable debugging logs and chat logs written to the local file system (text and mySQL respectively).  In other words, make the Ollama Client *ephemeral*.

## Limitations

This is a work in progress.  Limited testing has been done in Linux and Windows; no resources are available for macOS testing.

## License

The [Ollama desktop app source](https://github.com/ollama/ollama), [ollama-webchat-ubuntu](https://github.com/johnohhh1/ollama-webchat-ubuntu), and the source core here are MIT licensed.

