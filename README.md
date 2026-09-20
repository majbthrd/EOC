EOC (Ephemeral Ollama Client)
=============================

[Ollama](https://ollama.com) is a painless way to get a local LLM up and running.  It is multi-platform, it just works, and it is [open-source](https://github.com/ollama/ollama).

For Windows and macOS, Ollama even provides a multi-turn LLM chat GUI with a rich feature set (particularly in rendering).  An unofficial Linux adaptation is enabled by [this excellent effort](https://github.com/johnohhh1/ollama-webchat-ubuntu).

The great thing about open-source is that the software can be customized to suit the user's needs, and those changes can be shared back to the community.  The changes I made here were: a) disable Ollama cloud telemetry and downloads, and b) disable debugging logs and chat logs written to the local file system (text and mySQL respectively).  In other words, make the Ollama Client *ephemeral*.

## Limitations

This is a work in progress.  Limited testing has been done in Linux and Windows; no resources are available for macOS testing.

## Building

### Linux Prerequisites

```bash
sudo apt install build-essentials libwebkit2gtk-4.1-0 libgtk-3-0 zenity libwebkit2gtk-4.1-dev libgtk-3-dev golang-go nodejs npm
```
### Windows Prerequisites

Download and install [Go Compiler](https://go.dev/doc/install).

Download and install [Node.js](https://nodejs.org/en/download).

Download and install [MinGW-W64](https://www.mingw-w64.org/downloads/) (MinGW-W64 is needed to compile the Webview library; without this, the Go compiler gives a cryptic and unhelpful "undefined: webview" error).

### Common to all OSes

Download and extract the source code to a "EOC" subdirectory.  One way of doing this is:

```
git clone https://github.com/majbthrd/EOC.git
```

### Linux Build


```bash
cd EOC
bash build.sh
```

### Windows Build

```
cd EOC
build.bat
```

## License

The [Ollama desktop app source](https://github.com/ollama/ollama), [ollama-webchat-ubuntu](https://github.com/johnohhh1/ollama-webchat-ubuntu), and the source core here are MIT licensed.

