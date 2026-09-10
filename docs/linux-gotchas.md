## Shimmering Pixels

If you find that pixels seem to shimmer, try one of the following prefixes:

```
WEBKIT_DISABLE_DMABUF_RENDERER=1 ./EOC
```

```
WEBKIT_DISABLE_COMPOSITING_MODE=1 ./EOC
```

Apparently, there is a known incompatiblity between webkit and some graphics cards (particularly integrated Intel ones).

## Lock-up

If you find that, during LLM output rendering, your PC locks up with one CPU core using 100% to run webkit and another CPU core using 100% for systemd, try running the following before running EOC (or ollama-webchat-ubuntu):

```
export JavaScriptCoreUseJIT=0
export JSC_useJIT=0
```

This is apparently a known issue with webkit.  Deployed code *assumes* a modern CPU with AVX instructions, and these instructions are viewed as illegal by older CPUs (resulting in the described behavior).

